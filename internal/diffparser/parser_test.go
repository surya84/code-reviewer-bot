package diffparser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {
	sampleDiff := `diff --git a/main.go b/main.go
index 123..456 100644
--- a/main.go
+++ b/main.go
@@ -10,3 +10,4 @@
 
 func main() {
+	println("hello world")
 }
diff --git a/README.md b/README.md
index abc..def 100644
--- a/README.md
+++ b/README.md
@@ -1,2 +1,2 @@
 # My Project
-This is a test.
+This is a test project.`

	chunks := Parse(sampleDiff)
	assert.Len(t, chunks, 2)
	assert.Equal(t, "main.go", chunks[0].FilePath)
	assert.Equal(t, 10, chunks[0].StartLineNew)
	assert.Equal(t, "README.md", chunks[1].FilePath)
	assert.Equal(t, 1, chunks[1].StartLineNew)
}
