/*
 *     Copyright 2020 The Dragonfly Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package peer

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/go-http-utils/headers"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/time/rate"

	commonv1 "d7y.io/api/pkg/apis/common/v1"
	schedulerv1 "d7y.io/api/pkg/apis/scheduler/v1"

	"d7y.io/dragonfly/v2/client/config"
	"d7y.io/dragonfly/v2/client/daemon/metrics"
	"d7y.io/dragonfly/v2/client/daemon/storage"
	logger "d7y.io/dragonfly/v2/internal/dflog"
	"d7y.io/dragonfly/v2/pkg/idgen"
	"d7y.io/dragonfly/v2/pkg/net/http"
)

type StreamTaskRequest struct {
	// universal resource locator for different kind of storage
	URL string
	// url meta info
	URLMeta *commonv1.UrlMeta
	// http range
	Range *http.Range
	// peer's id and must be global uniqueness
	PeerID string
	//
	NeedPieceDownloadStatusChan bool
}

// StreamTask represents a peer task with stream io for reading directly without once more disk io
type StreamTask interface {
	// Start starts the special peer task, return an io.Reader for stream io
	// when all data transferred, reader return an io.EOF
	// attribute stands some extra data, like HTTP response Header
	Start(ctx context.Context) (rc io.ReadCloser, attribute map[string]string, err error)
}

type streamTask struct {
	*logger.SugaredLoggerOnWith
	ctx               context.Context
	span              trace.Span
	peerTaskConductor *peerTaskConductor
	pieceCh           chan *PieceInfo
	reserveBuf        []byte
	reserveIdx        int
}

func (ptm *peerTaskManager) newStreamTask(
	ctx context.Context,
	request *schedulerv1.PeerTaskRequest,
	rg *http.Range) (*streamTask, error) {
	metrics.StreamTaskCount.Add(1)
	var limit = rate.Inf
	if ptm.PerPeerRateLimit > 0 {
		limit = ptm.PerPeerRateLimit
	}

	// prefetch parent request
	var parent *peerTaskConductor
	if ptm.enabledPrefetch(rg) {
		parent = ptm.prefetchParentTask(request, "")
	}

	taskID := idgen.TaskIDV1(request.Url, request.UrlMeta)
	ptc, err := ptm.getPeerTaskConductor(ctx, taskID, request, limit, parent, rg, "", false)
	if err != nil {
		return nil, err
	}

	ctx, span := tracer.Start(ctx, config.SpanStreamTask, trace.WithSpanKind(trace.SpanKindClient))
	pt := &streamTask{
		SugaredLoggerOnWith: ptc.SugaredLoggerOnWith,
		ctx:                 ctx,
		span:                span,
		peerTaskConductor:   ptc,
		pieceCh:             ptc.broker.Subscribe(),
		reserveIdx:          -1,
	}
	return pt, nil
}

func (s *streamTask) Start(ctx context.Context) (io.ReadCloser, map[string]string, error) {
	// wait first piece to get content length and attribute (eg, response header for http/https)
	var firstPiece *PieceInfo

	attr := map[string]string{}
	attr[config.HeaderDragonflyTask] = s.peerTaskConductor.taskID
	attr[config.HeaderDragonflyPeer] = s.peerTaskConductor.peerID
	select {
	case <-ctx.Done():
		s.Errorf("%s", ctx.Err())
		s.span.RecordError(ctx.Err())
		s.span.End()
		return nil, attr, ctx.Err()
	case <-s.peerTaskConductor.failCh:
		err := s.peerTaskConductor.getFailedError()
		s.Errorf("wait first piece failed due to %s", err.Error())
		return nil, attr, err
	case <-s.peerTaskConductor.successCh:
		if s.peerTaskConductor.GetContentLength() != -1 {
			attr[headers.ContentLength] = fmt.Sprintf("%d", s.peerTaskConductor.GetContentLength())
		} else {
			attr[headers.TransferEncoding] = "chunked"
		}
		exa, err := s.peerTaskConductor.storage.GetExtendAttribute(ctx, nil)
		if err != nil {
			s.Errorf("read extend attribute error due to %s ", err.Error())
			return nil, attr, err
		}
		if exa != nil {
			for k, v := range exa.Header {
				attr[k] = v
			}
		}
		rc, err := s.peerTaskConductor.StorageManager.ReadAllPieces(
			ctx,
			&storage.ReadAllPiecesRequest{
				PeerTaskMetadata: storage.PeerTaskMetadata{
					PeerID: s.peerTaskConductor.peerID,
					TaskID: s.peerTaskConductor.taskID,
				}})
		return rc, attr, err
	case first := <-s.pieceCh:
		firstPiece = first
		exa, err := s.peerTaskConductor.storage.GetExtendAttribute(ctx, nil)
		if err != nil {
			s.Errorf("read extend attribute error due to %s ", err.Error())
			return nil, attr, err
		}
		if exa != nil {
			for k, v := range exa.Header {
				attr[k] = v
			}
		}
	}

	if s.peerTaskConductor.GetContentLength() != -1 {
		attr[headers.ContentLength] = fmt.Sprintf("%d", s.peerTaskConductor.GetContentLength())
	} else {
		attr[headers.TransferEncoding] = "chunked"
	}

	pr, pw := io.Pipe()
	var readCloser io.ReadCloser = pr
	go s.writeToPipe(firstPiece, pw)

	return readCloser, attr, nil
}

func (s *streamTask) writeOnePiece(w io.Writer, pieceNum int32, reserved bool) (int64, error) {
	pr, pc, err := s.peerTaskConductor.GetStorage().ReadPiece(s.ctx, &storage.ReadPieceRequest{
		PeerTaskMetadata: storage.PeerTaskMetadata{
			PeerID: s.peerTaskConductor.peerID,
			TaskID: s.peerTaskConductor.taskID,
		},
		PieceMetadata: storage.PieceMetadata{
			Num: pieceNum,
		},
	})
	if err != nil {
		return 0, err
	}

	if reserved {
		_, err := pr.Read(s.reserveBuf)
		if err != nil {
			s.reserveIdx = -1
			pc.Close()
			return 0, err
		}
		s.reserveIdx = 0
		return 0, pc.Close()
	}

	n, err := io.Copy(w, pr)
	if err != nil {
		pc.Close()
		return n, err
	}
	return n, pc.Close()
}

func (s *streamTask) writeToPipe(firstPiece *PieceInfo, pw *io.PipeWriter) {
	defer func() {
		s.span.End()
		s.reserveIdx = -1
	}()
	var (
		desired int32
		piece   *PieceInfo
		err     error
	)
	piece = firstPiece

	const (
		PieceCacheSize = 1024 * 1024 * 1
	)

	rangeSize := int(piece.RangeSize)
	if piece.RangeSize > PieceCacheSize {
		s.reserveBuf = make([]byte, piece.RangeSize)
	}

	tickTimer := time.NewTicker(5 * time.Second)
	for {
		if desired == piece.Num || desired <= piece.OrderedNum {
			if s.reserveIdx != -1 {
				reader := bytes.NewReader(s.reserveBuf[s.reserveIdx:])
				_, err := io.Copy(pw, reader)
				if err != nil {
					return
				}

				s.reserveIdx = -1
			}

			desired, err = s.writeOrderedPieces(desired, piece, pw)
			if err != nil {
				return
			}
		}

		select {
		case piece = <-s.pieceCh:
			continue
		case <-s.peerTaskConductor.successCh:
			if s.reserveIdx != -1 {
				reader := bytes.NewReader(s.reserveBuf[s.reserveIdx:])
				_, err := io.Copy(pw, reader)
				if err != nil {
					return
				}
			}

			s.writeRemainingPieces(desired, pw)
			return
		case <-s.ctx.Done():
			err = fmt.Errorf("context done due to: %s", s.ctx.Err())
			s.Errorf(err.Error())
			s.closeWithError(pw, err)
			return
		case <-s.peerTaskConductor.failCh:
			err = fmt.Errorf("stream close with peer task fail: %d/%s",
				s.peerTaskConductor.failedCode, s.peerTaskConductor.failedReason)
			s.Errorf(err.Error())
			s.closeWithError(pw, err)
			return
		case <-tickTimer.C:
			if s.reserveIdx != -1 {
				readLen := len(s.reserveBuf) / 20
				reader := bytes.NewReader(s.reserveBuf[s.reserveIdx : s.reserveIdx+readLen])
				_, err := io.Copy(pw, reader)
				if err != nil {
					return
				}

				s.reserveIdx += readLen
				if s.reserveIdx >= rangeSize {
					s.reserveIdx = -1
				}

				s.Debugf("wrote reserve piece %d to pipe, size %d", desired, readLen)
				continue
			}

		}
	}
}

func (s *streamTask) writeOrderedPieces(desired int32, piece *PieceInfo, pw *io.PipeWriter) (int32, error) {
	for {
		_, span := tracer.Start(s.ctx, config.SpanWriteBackPiece)
		span.SetAttributes(config.AttributePiece.Int(int(desired)))
		wrote, err := s.writeOnePiece(pw, desired, !piece.Finished && desired == piece.OrderedNum && len(s.reserveBuf) > 0)
		if err != nil {
			span.RecordError(err)
			span.End()
			s.Errorf("write to pipe error: %s", err)
			_ = pw.CloseWithError(err)
			return desired, err
		}
		span.SetAttributes(config.AttributePieceSize.Int(int(wrote)))
		s.Debugf("wrote piece %d to pipe, size %d", desired, wrote)
		span.End()

		desired++
		if desired > piece.OrderedNum {
			break
		}
	}
	return desired, nil
}

func (s *streamTask) writeRemainingPieces(desired int32, pw *io.PipeWriter) {
	for {
		// all data wrote to local storage, and all data wrote to pipe write
		if s.peerTaskConductor.readyPieces.Settled() == desired {
			s.Debugf("all %d pieces wrote to pipe", desired)
			pw.Close()
			return
		}
		_, span := tracer.Start(s.ctx, config.SpanWriteBackPiece)
		span.SetAttributes(config.AttributePiece.Int(int(desired)))
		wrote, err := s.writeOnePiece(pw, desired, false)
		if err != nil {
			span.RecordError(err)
			span.End()
			s.span.RecordError(err)
			s.Errorf("write to pipe error: %s", err)
			_ = pw.CloseWithError(err)
			return
		}
		span.SetAttributes(config.AttributePieceSize.Int(int(wrote)))
		span.End()
		s.Debugf("wrote piece %d to pipe, size %d", desired, wrote)
		desired++
	}
}

func (s *streamTask) closeWithError(pw *io.PipeWriter, err error) {
	s.Error(err)
	s.span.RecordError(err)
	if err = pw.CloseWithError(err); err != nil {
		s.Errorf("CloseWithError failed: %s", err)
	}
}
