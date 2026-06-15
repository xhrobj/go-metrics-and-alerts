package buildinfo

import (
	"fmt"
	"io"
)

// Print выводит информацию о сборке в writer.
// Пустые значения заменяются на N/A.
func Print(w io.Writer, version, date, commit string) error {
	if _, err := fmt.Fprintf(w, "Build version: %s\n", value(version)); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "Build date: %s\n", value(date)); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "Build commit: %s\n", value(commit)); err != nil {
		return err
	}

	return nil
}

func value(v string) string {
	const notAvailable = "N/A"

	if v == "" {
		return notAvailable
	}

	return v
}
