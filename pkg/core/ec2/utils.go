package ec2

func convertPortsToStringMap(portsList []int32) map[string]bool {
	m := make(map[string]bool)

	for _, port := range portsList {
		if name, ok := reverseAllowedRules[port]; ok {
			m[name] = true
		}
	}

	return m
}
