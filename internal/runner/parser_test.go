package runner

import (
	"testing"
)

func TestParseOutputGo(t *testing.T) {
	output := `
# tide/cmd/tide
main.go:4:5: undefined: fmt.Println
./sub/worker.go:18:10: cannot use val (variable of type int) as string in argument to log
`
	diags := ParseOutput(output)
	if len(diags) != 2 {
		t.Fatalf("expected 2 diagnostics, got %d", len(diags))
	}

	if diags[0].File != "main.go" || diags[0].Line != 4 || diags[0].Col != 5 || diags[0].Severity != "error" {
		t.Errorf("unexpected diag[0]: %+v", diags[0])
	}
	if diags[0].Message != "undefined: fmt.Println" {
		t.Errorf("unexpected message: %s", diags[0].Message)
	}

	if diags[1].File != "sub/worker.go" || diags[1].Line != 18 || diags[1].Col != 10 {
		t.Errorf("unexpected diag[1]: %+v", diags[1])
	}
}

func TestParseOutputC(t *testing.T) {
	output := `
main.c:12:4: error: 'foo' undeclared (first use in this function)
main.c:15:9: warning: unused variable 'bar' [-Wunused-variable]
src/util.c:20:1: fatal error: stdio.h: No such file or directory
`
	diags := ParseOutput(output)
	if len(diags) != 3 {
		t.Fatalf("expected 3 diagnostics, got %d", len(diags))
	}

	if diags[0].Severity != "error" || diags[0].Line != 12 || diags[0].Col != 4 {
		t.Errorf("unexpected diag[0]: %+v", diags[0])
	}
	if diags[1].Severity != "warning" || diags[1].Line != 15 {
		t.Errorf("unexpected diag[1]: %+v", diags[1])
	}
	if diags[2].Severity != "error" || diags[2].Line != 20 || diags[2].File != "src/util.c" {
		t.Errorf("unexpected diag[2]: %+v", diags[2])
	}
}

func TestMatchesFile(t *testing.T) {
	if !MatchesFile("main.go", "main.go") {
		t.Errorf("expected match for main.go and main.go")
	}
	if !MatchesFile("main.go", "/path/to/main.go") {
		t.Errorf("expected match for main.go and /path/to/main.go")
	}
	if !MatchesFile("src/main.c", "/root/src/main.c") {
		t.Errorf("expected match for src/main.c and /root/src/main.c")
	}
	if MatchesFile("other.go", "main.go") {
		t.Errorf("did not expect match for other.go and main.go")
	}
}
