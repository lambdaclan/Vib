package core

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// BuildAptModule builds a module that installs packages
// using the apt package manager
func BuildAptModule(recipe *Recipe, module Module) (string, error) {
	if len(module.Source.Packages) > 0 {
		packages := ""
		for _, pkg := range module.Source.Packages {
			packages += pkg + " "
		}

		return fmt.Sprintf("apt install -y %s && apt clean", packages), nil
	}

	if len(module.Source.Paths) > 0 {
		cmd := ""

		for _, path := range module.Source.Paths {
			instPath := filepath.Join(recipe.ParentPath, path+".inst")
			pkgs := ""
			file, err := os.Open(instPath)
			if err != nil {
				return "", err
			}
			defer file.Close()

			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				pkgs += scanner.Text() + " "
			}

			if err := scanner.Err(); err != nil {
				return "", err
			}

			cmd += fmt.Sprintf("apt install -y %s && ", pkgs)

		}

		cmd += "apt clean"
		return cmd, nil
	}

	return "", errors.New("no packages or paths specified")
}
