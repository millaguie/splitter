package cli

import "log/slog"

func InstanceField(id int) slog.Attr {
	return slog.Int("instance", id)
}

func CountryField(code string) slog.Attr {
	return slog.String("country", code)
}

func PortField(port int) slog.Attr {
	return slog.Int("port", port)
}
