package porcelain

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
	"github.com/netlify/open-api/go/plumbing/operations"
	"github.com/stretchr/testify/require"
)

func TestGetLFSSha(t *testing.T) {
	t.Run("test with not a pointer file", func(t *testing.T) {
		file := strings.NewReader("Not a pointer file")
		data, err := readLFSData(file)
		if err != nil {
			t.Fatal(err)
		}

		if data != nil {
			t.Fatal("expected data to be nil without proper formatting")
		}
	})

	t.Run("test with v1 pointer", func(t *testing.T) {
		content := `version https://git-lfs.github.com/spec/v1
oid sha256:7e56e498ccb4cbb9c672e1aed6710fb91b2fd314394a666c11c33b2059ea3d71
size 1743570
`
		file := strings.NewReader(content)
		data, err := readLFSData(file)
		if err != nil {
			t.Fatal(err)
		}

		if data.SHA != "7e56e498ccb4cbb9c672e1aed6710fb91b2fd314394a666c11c33b2059ea3d71" {
			t.Fatalf("expected `7e56e498ccb4cbb9c672e1aed6710fb91b2fd314394a666c11c33b2059ea3d71`, got `%v`", data.SHA)
		}

		if data.Size != 1743570 {
			t.Fatalf("expected `1743570`, got `%v`", data.Size)
		}
	})
}

func TestAddWithLargeMedia(t *testing.T) {
	files := newDeployFiles()
	tests := []struct {
		rel string
		sum string
	}{
		{"foo.jpg", "sum1"},
		{"bar.jpg", "sum2"},
		{"baz.jpg", "sum3:originalsha"},
	}

	for _, test := range tests {
		file := &FileBundle{}
		file.Sum = test.sum
		files.Add(test.rel, file)
	}

	out := files.Hashed["sum3"]
	if len(out) != 1 {
		t.Fatalf("expected `%d`, got `%d`", 1, len(out))
	}
	out2 := files.Sums["baz.jpg"]
	if out2 != "sum3:originalsha" {
		t.Fatalf("expected `%v`, got `%v`", "sum3:originalsha", out2)
	}
}

func TestUploadDeployFileWithTimeoutError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(http.StatusCreated)
		rw.Write([]byte("test result"))
	}))
	defer server.Close()

	httpClient := http.DefaultClient
	authInfo := runtime.ClientAuthInfoWriterFunc(func(r runtime.ClientRequest, _ strfmt.Registry) error {
		_ = r.SetHeaderParam("User-Agent", "buildbot")
		_ = r.SetHeaderParam("Authorization", "Bearer 1234")
		return nil
	})

	endpoint, _ := url.Parse(server.URL)
	tr := apiClient.NewWithClient(endpoint, "/api/v1", []string{"http"}, httpClient)
	client := NewRetryable(tr, strfmt.Default, 1)

	params := operations.NewUploadDeployFileParams()
	resp, operationError := client.Operations.UploadDeployFunction(params, authInfo)

	require.NoError(t, operationError)
}
