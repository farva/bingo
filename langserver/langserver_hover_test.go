package langserver

import (
	"context"
	"fmt"
	"github.com/sourcegraph/go-langserver/langserver/util"
	"log"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sourcegraph/go-langserver/pkg/lsp"
	"github.com/sourcegraph/jsonrpc2"
)


func TestHover(t *testing.T) {
	test := func(t *testing.T, pkgDir string, input string, output string) {
		testHover(t, &hoverTestCase{pkgDir: pkgDir, input: input, output: output})
	}

	t.Run("basic hover", func(t *testing.T) {
		test(t, basicPkgDir, "a.go:1:9", "package p")
		test(t, basicPkgDir, "a.go:1:17", "func A()")
		test(t, basicPkgDir, "a.go:1:23",  "func A()")
		test(t, basicPkgDir, "b.go:1:17", "func B()")
		test(t, basicPkgDir, "b.go:1:23", "func A()")
	})

	t.Run("detailed hover", func(t *testing.T) {
		test(t, detailedPkgDir, "a.go:1:28", "struct field F string")
		test(t, detailedPkgDir, "a.go:1:17", `type T struct; struct {
    F string
}`)
	})

	t.Run("xtest hover", func(t *testing.T) {
		test(t, xtestPkgDir, "a.go:1:16", "var A int")
		test(t, xtestPkgDir, "x_test.go:1:40", "var X int")
		test(t, xtestPkgDir, "x_test.go:1:46", "var A int")
		test(t, xtestPkgDir, "a_test.go:1:16", "var X int")
		test(t, xtestPkgDir, "a_test.go:1:20", "var A int")
	})

	t.Run("test hover", func(t *testing.T) {
		test(t, testPkgDir, "a_test.go:1:37", "var X int")
		test(t, testPkgDir, "a_test.go:1:43", "var B int")
	})

	t.Run("subdirectory hover", func(t *testing.T) {
		test(t, subdirectoryPkgDir, "a.go:1:17",    "func A()")
		test(t, subdirectoryPkgDir, "a.go:1:23",    "func A()")
		test(t, subdirectoryPkgDir, "d2/b.go:1:98", "func B()")
		test(t, subdirectoryPkgDir, "d2/b.go:1:106", "func A()")
		test(t, subdirectoryPkgDir, "d2/b.go:1:111", "func B()")
	})

	t.Run("multiple packages in dir", func(t *testing.T) {
		test(t, multiplePkgDir, "a.go:1:17", "func A()")
		test(t, multiplePkgDir, "a.go:1:23", "func A()")
	})

	t.Run("goroot", func(t *testing.T) {
		test(t, gorootPkgDir, "a.go:1:40", "func Println(a ...interface{}) (n int, err error); Println formats using the default formats for its operands and writes to standard output. Spaces are always added between operands and a newline is appended. It returns the number of bytes written and any write error encountered. \n\n")
	})
}

type hoverTestCase struct {
	pkgDir string
	input  string
	output string
}

func testHover(tb testing.TB, c *hoverTestCase) {
	tbRun(tb, fmt.Sprintf("hover-%s", strings.Replace(c.input, "/", "-", -1)), func(t testing.TB) {
		dir, err := filepath.Abs(c.pkgDir)
		if err != nil {
			log.Fatal("testHover", err)
		}
		doHoverTest(t, ctx, conn, util.PathToURI(dir), c.input, c.output)
	})
}

func doHoverTest(t testing.TB, ctx context.Context, conn *jsonrpc2.Conn, rootURI lsp.DocumentURI, pos, want string) {
	file, line, char, err := parsePos(pos)
	if err != nil {
		t.Fatal(err)
	}
	hover, err := callHover(ctx, conn, uriJoin(rootURI, file), line, char)
	if err != nil {
		t.Fatal(err)
	}
	if hover != want {
		t.Fatalf("got %q, want %q", hover, want)
	}
}

func callHover(ctx context.Context, c *jsonrpc2.Conn, uri lsp.DocumentURI, line, char int) (string, error) {
	var res struct {
		Contents markedStrings `json:"contents"`
		lsp.Hover
	}
	err := c.Call(ctx, "textDocument/hover", lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{URI: uri},
		Position:     lsp.Position{Line: line, Character: char},
	}, &res)
	if err != nil {
		return "", err
	}
	var str string
	for i, ms := range res.Contents {
		if i != 0 {
			str += "; "
		}
		str += ms.Value
	}
	return str, nil
}