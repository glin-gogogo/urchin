package training

type DataOptions struct {
	// maxBufferLine capacity of lines which reads from record once.
	MaxBufferLine int

	// maxRecordLine capacity of lines which local memory obtains.
	MaxRecordLine int
}

type DataOptionFunc func(options *DataOptions)

func WithMaxBufferLine(MaxBufferLine int) DataOptionFunc {
	return func(options *DataOptions) {
		options.MaxBufferLine = MaxBufferLine
	}
}

func WithMaxRecordLine(MaxRecordLine int) DataOptionFunc {
	return func(options *DataOptions) {
		options.MaxRecordLine = MaxRecordLine
	}
}
