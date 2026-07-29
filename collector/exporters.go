package main

func exporterAllowed(ip string) bool {
	c := GetConfig()
	list := c.Exporters
	if len(list) == 0 {
		return ip == SourceIP
	}
	for _, e := range list {
		if e == ip {
			return true
		}
	}
	return false
}
