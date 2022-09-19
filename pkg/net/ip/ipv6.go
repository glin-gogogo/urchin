/*
 *     Copyright 2022 The Dragonfly Authors
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

package ip

import (
	"fmt"

	logger "d7y.io/dragonfly/v2/internal/dflog"
)

var IPv6 string

const (
	internalIPv6 = "::1"
)

func init() {
	ip, err := externalIPv6()
	if err != nil {
		logger.Debugf("Failed to get IPv6 address: %s", err.Error())
		logger.Infof("Use %s as IPv6 addr", internalIPv6)
		IPv6 = internalIPv6
	} else {
		IPv6 = ip
	}
}

// externalIPv6 returns the available IPv6.
func externalIPv6() (string, error) {
	ips, err := ipAddrs()
	if err != nil {
		return "", err
	}

	var externalIPs []string
	for _, ip := range ips {
		v4 := ip.To4()
		if v4 != nil {
			continue // skip all ipv4 address
		}
		ip = ip.To16()
		externalIPs = append(externalIPs, ip.String())
	}

	if len(externalIPs) == 0 {
		return "", fmt.Errorf("can not found external ipv4")
	}

	return externalIPs[0], nil
}
