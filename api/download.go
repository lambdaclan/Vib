package api

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// retrieves the Source directory of a given Source
func GetSourcePath(source Source, moduleName string) string {
	if source.Type == "git" {
		var dest string
		if source.Destination != "" {
			repoName := strings.Split(source.URL, "/")
			dest = filepath.Join(moduleName, strings.ReplaceAll(repoName[len(repoName)-1], ".git", ""))
		} else {
			dest = filepath.Join(moduleName, source.Destination)
		}
		return dest
	} else if source.Type == "tar" {
		return filepath.Join(moduleName, source.Destination)
	}
	return ""
}

// DownloadSource downloads a source to the downloads directory
// according to its type (git, tar, ...)
func DownloadSource(downloadPath string, source Source, moduleName string) error {
	fmt.Printf("Downloading source: %s\n", source.URL)

	if source.Type == "git" {
		return DownloadGitSource(downloadPath, source, moduleName)
	} else if source.Type == "tar" {
		err := DownloadTarSource(downloadPath, source, moduleName)
		if err != nil {
			return err
		}
		return checksumValidation(source, filepath.Join(downloadPath, GetSourcePath(source, moduleName), moduleName+".tar"))
	} else {
		return fmt.Errorf("unsupported source type %s", source.Type)
	}
}

// DownloadGitSource downloads a git source to the downloads directory
// and checks out the commit or tag
func DownloadGitSource(downloadPath string, source Source, moduleName string) error {
	fmt.Printf("Downloading git source: %s\n", source.URL)

	if source.Commit == "" && source.Tag == "" && source.Branch == "" {
		return fmt.Errorf("missing source commit, tag or branch")
	}

	dest := filepath.Join(downloadPath, GetSourcePath(source, moduleName))
	os.MkdirAll(dest, 0777)

	if source.Tag != "" {
		fmt.Printf("Using tag %s\n", source.Tag)

		cmd := exec.Command(
			"git",
			"clone", source.URL,
			"--depth", "1",
			"--branch", source.Tag,
			dest,
		)
		err := cmd.Run()
		if err != nil {
			return err
		}
	} else {
		fmt.Printf("Using commit %s\n", source.Commit)

		if source.Branch == "" {
			return fmt.Errorf("missing source branch")
		}

		fmt.Printf("Cloning repository: %s\n", source.URL)
		cmd := exec.Command(
			"git",
			"clone", source.URL,
			dest,
		)
		err := cmd.Run()
		if err != nil {
			return err
		}

		if source.Commit == "latest" {
			cmd := exec.Command(
				"git", "--no-pager", "log", "-n", "1", "--pretty=format:\"%H\"", source.Branch,
			)
			cmd.Dir = dest
			latest_tag, err := cmd.Output()
			if err != nil {
				return err
			}
			source.Commit = strings.Trim(string(latest_tag), "\"")
		}

		fmt.Printf("Checking out branch: %s\n", source.Branch)
		cmd = exec.Command(
			"git",
			"checkout",
			"-B", source.Branch,
		)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Dir = dest
		err = cmd.Run()
		if err != nil {
			return err
		}

		fmt.Printf("Resetting to commit: %s\n", source.Commit)
		cmd = exec.Command(
			"git",
			"reset", "--hard", source.Commit,
		)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Dir = dest
		err = cmd.Run()
		if err != nil {
			return err
		}
	}

	return nil
}

// DownloadTarSource downloads a tar archive to the downloads directory
func DownloadTarSource(downloadPath string, source Source, moduleName string) error {
	fmt.Printf("Source is tar: %s\n", source.URL)
	//Create the destination path
	dest := filepath.Join(downloadPath, GetSourcePath(source, moduleName))
	os.MkdirAll(dest, 0777)
	//Download the resource
	res, err := http.Get(source.URL)
	if err != nil {
		return err
	}

	defer res.Body.Close()
	//Create the destination tar file
	file, err := os.Create(filepath.Join(dest, moduleName+".tar"))
	if err != nil {
		return err
	}
	//Close the file when the function ends
	defer file.Close()
	//Copy the response body to the destination file
	_, err = io.Copy(file, res.Body)
	if err != nil {
		return err
	}

	return nil
}

// MoveSources moves all sources from the downloads directory to the
// sources directory
func MoveSources(downloadPath string, sourcesPath string, sources []Source, moduleName string) error {
	fmt.Println("Moving sources")

	for _, source := range sources {
		err := MoveSource(downloadPath, sourcesPath, source, moduleName)
		if err != nil {
			return err
		}
	}

	return nil
}

// MoveSource moves a source from the downloads directory to the
// sources directory, by extracting if a tar archive or moving if a
// git repository
func MoveSource(downloadPath string, sourcesPath string, source Source, moduleName string) error {
	fmt.Printf("Moving source: %s\n", moduleName)

	if source.Type == "git" {
		dest := GetSourcePath(source, moduleName)
		return os.Rename(
			filepath.Join(downloadPath, dest),
			filepath.Join(sourcesPath, dest),
		)
	} else if source.Type == "tar" {
		os.MkdirAll(filepath.Join(sourcesPath, GetSourcePath(source, moduleName)), 0777)
		cmd := exec.Command(
			"tar",
			"-xf", filepath.Join(downloadPath, GetSourcePath(source, moduleName), moduleName+".tar"),
			"-C", filepath.Join(sourcesPath, GetSourcePath(source, moduleName)),
		)
		err := cmd.Run()
		if err != nil {
			return err
		}

		return os.Remove(filepath.Join(downloadPath, GetSourcePath(source, moduleName), moduleName+".tar"))
	} else {
		return fmt.Errorf("unsupported source type %s", source.Type)
	}
}

// checksumValidation validates the checksum of a file
func checksumValidation(source Source, path string) error {
	//No checksum provided
	if len(strings.TrimSpace(source.Checksum)) == 0 {
		return nil
	}
	//Open the file
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	//Close the file when the function ends
	defer file.Close()
	//Calculate the checksum
	checksum := sha256.New()
	_, err = io.Copy(checksum, file)
	if err != nil {
		return fmt.Errorf("could not calculate tar file checksum")
	}

	//Validate the checksum
	if fmt.Sprintf("%x", checksum.Sum(nil)) != source.Checksum {

		return fmt.Errorf("tar file checksum doesn't match")
	}

	return nil
}
