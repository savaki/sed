package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/urfave/cli/v2"
)

var opts struct {
	Filename string
	Edits    cli.StringSlice
}

func main() {
	app := cli.NewApp()
	app.Usage = "minimal implementation of sed"
	app.Description = "minimalistic implementation of sed for environments without libc"
	app.Action = action
	app.HideHelp = true
	app.Flags = []cli.Flag{
		&cli.StringSliceFlag{
			Name:        "e",
			Usage:       "appends sed style editing command",
			Destination: &opts.Edits,
		},
		&cli.StringFlag{
			Name:        "i",
			Usage:       "input file to edit",
			Destination: &opts.Filename,
			Required:    true,
		},
	}
	if err := app.Run(os.Args); err != nil {
		log.Fatalln(err)
	}
}

func action(_ *cli.Context) error {
	data, err := ioutil.ReadFile(opts.Filename)
	if err != nil {
		return fmt.Errorf("sed failed on file, %v: %w", opts.Filename, err)
	}

	data, err = editSubstitute(data, opts.Edits.Value()...)
	if err != nil {
		return err
	}

	data, err = editAppend(data, opts.Edits.Value()...)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(opts.Filename, data, 0644)
}

// editAppend mimics the sed append edit
func editAppend(data []byte, edits ...string) ([]byte, error) {
	data = append(data, '\n')

	reAppend := regexp.MustCompile(`/([^/]+)/\s*a(.*)`)
	r := bufio.NewReader(bytes.NewReader(data))
	var ss []string
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("failed to scan file, %v: %w", opts.Filename, err)
		}

		ss = append(ss, line)

		for _, edit := range edits {
			match := reAppend.FindStringSubmatch(edit)
			if len(match) == 0 {
				continue
			}
			if strings.HasPrefix(line, match[1]) {
				ss = append(ss, match[2]+"\n")
			}
		}
	}
	data = []byte(strings.Join(ss, ""))

	return data, nil
}

// editAppend mimics the sed substitution edit
func editSubstitute(data []byte, edits ...string) ([]byte, error) {
	reSubst := regexp.MustCompile(`^s(.)(.*)(.)$`)
	for _, edit := range edits {
		match := reSubst.FindStringSubmatch(edit)
		if len(match) == 0 {
			continue
		}

		parts := strings.SplitN(match[2], match[1], -1)
		if len(parts) != 2 {
			return nil, fmt.Errorf("sed failed: invalid edit body, %v", edit)
		}

		oldString, newString := parts[0], parts[1]
		data = bytes.ReplaceAll(data, []byte(oldString), []byte(newString))
	}
	return data, nil
}
