// Copyright (c) 2024, xplshn, u-root and contributors  [3BSD]
// For more details refer to https://github.com/xplshn/a-utils
package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

/* Considered harmful:
   If these are ever implemented... Its for compatibility reasons.
   	  -n Number output lines
   	  -b Number nonempty lines
   	  -v Show nonprinting characters as ^x or M-x
   	  -t ...and tabs as ^I
   	  -e ...and end lines with $
   	  -A Same as -vte
*/

func main() {
	syntaxHighlighting := flag.Bool("x", false, "enable syntax highlighting")
	consideredHarmful := flag.Bool("v", false, "cat -v considered harmful!")
	flag.Usage = func() {
		p := `
 Copyright (c) 2024, xplshn, u-root and contributors  [3BSD]
 For more details refer to https://github.com/xplshn/a-utils

  Description
    Concatenates files and prints them to stdout.
  Synopsis:
    cat <-x> [FILES]...
  Behavior:
    If no files are specified, read from stdin.
  Options:
    -x: enable syntax highlighting
  Note:
    This implementation of cat is NOT POSIX. It implements syntax highlighting using the -x parameter.
`
		fmt.Println(p)
	}
	flag.Parse()
	args := flag.Args()

	if *consideredHarmful {
		fmt.Println("cat -v considered harmful!")
		os.Exit(69)
	}

	if err := run(os.Stdin, os.Stdout, *syntaxHighlighting, args...); err != nil {
		fmt.Fprintln(os.Stderr, "cat failed:", err)
		os.Exit(1)
	}
}

func run(stdin io.Reader, stdout io.Writer, syntaxHighlighting bool, args ...string) error {
	if len(args) == 0 {
		return processInput(stdin, stdout, "stdin", syntaxHighlighting)
	}

	for _, file := range args {
		var reader io.Reader
		if file == "-" {
			reader = stdin
		} else {
			f, err := os.Open(file)
			if err != nil {
				return fmt.Errorf("failed to open file %s: %v", file, err)
			}
			defer f.Close()
			reader = f
		}
		if err := processInput(reader, stdout, file, syntaxHighlighting); err != nil {
			return err
		}
	}
	return nil
}

func processInput(reader io.Reader, writer io.Writer, fileName string, syntaxHighlighting bool) error {
	if syntaxHighlighting {
		return highlightCat(reader, writer, fileName)
	}
	return cat(reader, writer)
}

func cat(reader io.Reader, writer io.Writer) error {
	_, err := io.Copy(writer, reader)
	return err
}

func highlightCat(reader io.Reader, writer io.Writer, fileName string) error {
	lexer := lexers.Match(fileName)
	if lexer == nil {
		lexer = lexers.Fallback
	}
	lexer = chroma.Coalesce(lexer)

	style := styles.Get("swapoff")
	if style == nil {
		style = styles.Get(os.Getenv("A_SYHX_COLOR_SCHEME"))
	}
	if style == nil {
		style = styles.Fallback
	}

	formatterName := os.Getenv("A_SYHX_FORMAT")
	if formatterName == "" {
		formatterName = "terminal16"
	}
	formatter := formatters.Get(formatterName)
	if formatter == nil {
		formatter = formatters.Fallback
	}

	contents, err := io.ReadAll(reader)
	if err != nil {
		return err
	}

	iterator, err := lexer.Tokenise(nil, string(contents))
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	if err := formatter.Format(&buf, style, iterator); err != nil {
		return err
	}

	_, err = io.Copy(writer, &buf)
	return err
}