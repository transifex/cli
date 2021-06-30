package txlib

import (
	"io/ioutil"
	"os"
	"strings"

	"github.com/transifex/cli/internal/txlib/config"
)

// Takes a local language and converts it to a transifex language code
func getTxLanguageCode(
	languageMappings map[string]string,
	localLanguageCode string,
	cfgResource *config.Resource,
) string {
	reverseLanguageOverrides := make(map[string]string)

	for k, v := range languageMappings {
		reverseLanguageOverrides[v] = k
	}

	for k, v := range cfgResource.LanguageMappings {
		reverseLanguageOverrides[v] = k
	}

	if val, ok := reverseLanguageOverrides[localLanguageCode]; ok {
		return val
	}
	return localLanguageCode
}

const PathSeparator = string(os.PathSeparator)

/*
Recursively search under the directory 'root' for files that match the
'fileFilter'. The original file filter must have exactly one instance of
"<lang>" in it.

If nothing is found, an empty map will be returned.

If 'fileFilter' is empty, which means that we are in the last step of the
recursion, 'root' is returned if it exists in the filesystem and is a file.

If, in the current iteration, the file filter does not have "<lang>" (which
means that the "<lang>" part is now in the 'root'), then the matching file, if
found, will be returned under the "" key ({"": "/path/to/file.txt"}).

If the first item in 'fileFilter' does have a "<lang>" in it, then the contents
of the 'root' directory (it must be a directory) will be matched against the
pattern, the function will be called recursively for a new root that contains
the matched path and the result of the recursive function will be added to the
result of the current function with the matched language as the key.

Examples:

1. Assuming the filesystem looks like this:
    /path/to/root/
      |
      + file.txt

Then the invocation of:

    searchFileFilter("/path/to/root/file.txt", "")

will check that 'root' does exist and is not a directory and return:

    map[string]string{"": "/path/to/root/file.txt"}

The invocation of:

    searchFileFilter("/path/to/root", "file.txt")

will recursively call the previous invocation and return its result:

    map[string]string{"": "/path/to/root/file.txt"}

2. Assuming the filesystem looks like this:

    /path/to/root/
      |
      + en.txt
      |
      + fr.txt

The invocation of:

    searchFileFilter("/path/to/root/en.txt", "")

as before, will return:

    map[string]string{"": "/path/to/root/en.txt"}

But, the invocation of:

    searchFileFilter("/path/to/root", "<lang>.txt")

will inspect the contents of 'root', match the 2 files against the pattern,
make 2 invocations similar to the first (one with "en" in 'root' and one with
"fr") and return their results using the matched language codes as keys. So:

    map[string]string{"en": "/path/to/root/en.txt",
                      "fr": "/path/to/root/fr.txt"}

3. Finally, assuming the filesystem looks like this:

    /path/to/root/
      |
      + en/
      |   |
      |   + file.txt
      |
      + fr/
          |
          + file.txt

The following calls and results will happen:

    searchFileFilter("/path/to/root/en/file.txt", "")
    // map[string]string{"": "/path/to/root/en/file.txt"}

    searchFileFilter("/path/to/root/en", "file.txt")
    // map[string]string{"": "/path/to/root/en/file.txt"}

    searchFileFilter("/path/to/root", "<lang>/file.txt")
    // map[string]string{"en": "/path/to/root/en/file.txt",
                         "fr": "/path/to/root/fr/file.txt"}
*/
func searchFileFilter(root string, fileFilter string) map[string]string {
	result := make(map[string]string)

	fileFilterSlice := strings.Split(fileFilter, PathSeparator)

	if len(fileFilter) == 0 {
		fileInfo, err := os.Stat(root)
		if err != nil || fileInfo.IsDir() {
			return result
		}
		result[""] = root
		return result
	}

	if !strings.Contains(fileFilterSlice[0], "<lang>") {
		// Recursively go deeper
		newRoot := strings.Join([]string{root, fileFilterSlice[0]},
			PathSeparator)
		newFileFilter := strings.Join(fileFilterSlice[1:], PathSeparator)
		return searchFileFilter(newRoot, newFileFilter)
	} else {
		// Sometime before we checked that the original 'fileFilterSlice' had
		// exactly one "<lang>" in it, so 'parts' is guaranteed to be of size 2
		parts := strings.Split(fileFilterSlice[0], "<lang>")
		left := parts[0]
		right := parts[1]

		fileInfos, err := ioutil.ReadDir(root)
		if err != nil {
			return result
		}
		for _, fileInfo := range fileInfos {
			name := fileInfo.Name()
			if len(name) < len(left)+len(right) ||
				// before-fr-after
				// ^^^^^^^
				name[:len(left)] != left ||
				// before-fr-after
				//          ^^^^^^
				name[len(name)-len(right):] != right {
				continue
			}
			languageCode := name[len(left) : len(name)-len(right)]

			newRoot := strings.Join([]string{root, name}, PathSeparator)
			newFileFilter := strings.Join(fileFilterSlice[1:], PathSeparator)
			answer := searchFileFilter(newRoot, newFileFilter)

			path, exists := answer[""]
			if exists {
				result[languageCode] = path
			}
		}
		return result
	}
}
