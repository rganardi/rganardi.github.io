package main

import (
	"bytes"
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

var (
	pass        []byte
	outputFound = false
	htmlCmd     *string
	verbose     *bool
)

type Header interface {
	Get(key string) string
}

func decrypt(in io.Reader) (out io.Reader, err error) {
	cmd := exec.Command("gpg", "--decrypt")
	cmd.Stdin = in
	outBytes, err := cmd.Output()
	out = bytes.NewReader(outBytes)
	return
}

func decode(in io.Reader, cte string, charset string) (out io.Reader, err error) {
	switch cte {
	case "base64":
		out = base64.NewDecoder(base64.StdEncoding, in)
	case "quoted-printable":
		out = quotedprintable.NewReader(in)
	default:
		out = in
	}
	return
}

func showHeader(k string, verbose bool) bool {
	if verbose {
		return true
	}

	switch k {
	case "Date":
		fallthrough
	case "From":
		fallthrough
	case "To":
		fallthrough
	case "Subject":
		return true
	}

	return false
}

func handleMessage(body io.Reader, h Header) (err error) {
	defer func() {
		if err != nil {
			_, f, l, _ := runtime.Caller(1)
			err = fmt.Errorf("%s:%d: %w", f, l, err)
		}
		return
	}()

	mediaType, params, err := mime.ParseMediaType(h.Get("Content-Type"))
	if err != nil {
		return err
	}

	cte := h.Get("Content-Transfer-Encoding")

	switch mediaType {
	case "multipart/alternative":
		// TODO: Only handle one alternative
		fallthrough
	case "multipart/related":
		err = handleMultipartRelated(body, params["boundary"])
	case "multipart/encrypted":
		err = handleMultipartEncrypted(body, params["protocol"])
	case "text/plain":
		err = handleTextPlain(body, cte, params["charset"])
	case "text/html":
		err = handleTextHtml(body, cte, params["charset"])
	}
	if err != nil {
		return err
	}

	return
}

func handleMultipartEncrypted(r io.Reader, protocol string) (err error) {
	if protocol != "application/pgp-encrypted" {
		err = fmt.Errorf("unknown protocol")
		return
	}
	decrypted, err := decrypt(r)
	if err != nil {
		return
	}
	m, err := mail.ReadMessage(decrypted)
	if err != nil {
		return err
	}

	err = handleMessage(m.Body, m.Header)
	if err != nil {
		return err
	}

	return
}

func handleMultipartRelated(r io.Reader, boundary string) (err error) {
	defer func() {
		if err != nil {
			_, f, l, _ := runtime.Caller(1)
			err = fmt.Errorf("%s:%d: %w", f, l, err)
		}
		return
	}()
	multipartReader := multipart.NewReader(r, boundary)

	for {
		ent, err := multipartReader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		defer ent.Close()

		err = handleMessage(ent, ent.Header)
		if err != nil {
			return err
		}
	}

	return
}

func handleTextPlain(r io.Reader, cte string, charset string) (err error) {
	decoded, err := decode(r, cte, charset)
	if err != nil {
		return err
	}
	io.Copy(os.Stdout, decoded)
	outputFound = true

	return
}

func handleTextHtml(r io.Reader, cte string, charset string) (err error) {
	bin := strings.Split(*htmlCmd, " ")[0]
	args := strings.Split(*htmlCmd, " ")[1:]
	cmd := exec.Command(bin, args...)

	decoded, err := decode(r, cte, charset)
	if err != nil {
		return err
	}

	cmd.Stdin = decoded
	out, err := cmd.Output()
	if err != nil {
		return err
	}
	fmt.Printf("%s\n", string(out))
	outputFound = true

	return
}

func parseMessage(entity *mail.Message) (err error) {
	defer func() {
		if err != nil {
			_, f, l, _ := runtime.Caller(1)
			err = fmt.Errorf("%s:%d: %w", f, l, err)
		}
		return
	}()

	err = handleMessage(entity.Body, entity.Header)
	if err != nil {
		return err
	}

	// ignore io.EOF
	if err == io.EOF {
		return nil
	}

	return
}

func run() (err error) {
	defer func() {
		if err != nil {
			_, f, l, _ := runtime.Caller(1)
			err = fmt.Errorf("%s:%d: %w", f, l, err)
		}
		return
	}()

	htmlCmd = flag.String("htmlcmd", "w3m -T text/html", "html command")
	verbose = flag.Bool("v", false, "verbose")
	flag.Parse()

	mailBuf, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		return
	}

	// ignore 'From ' from RFC822
	if bytes.HasPrefix(mailBuf, []byte("From ")) {
		firstLine := bytes.IndexRune(mailBuf, '\n')
		mailBuf = mailBuf[firstLine+1:]
	}

	m, err := mail.ReadMessage(bytes.NewReader(mailBuf))
	if err != nil {
		return err
	}

	for k, v := range m.Header {
		if showHeader(k, *verbose) {
			fmt.Printf("%v: %v\n", k, strings.Join(v, " "))
		}
	}

	fmt.Println()

	err = parseMessage(m)
	if err != nil {
		return
	}

	if !outputFound {
		return fmt.Errorf("no text/plain part found")
	}

	return
}

func main() {
	err := run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %#v\n", err)
		os.Exit(1)
	}
	return
}
