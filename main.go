// +build OMIT

package main

import (
	"bytes"
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"github.com/docopt/docopt-go"
)

// A result is the product of reading and summing a file using MD5.
type result struct {
	path string
	sum  [md5.Size]byte
	err  error
}

// sumFiles starts goroutines to walk the directory tree at root and digest each
// regular file.  These goroutines send the results of the digests on the result
// channel and send the result of the walk on the error channel.  If done is
// closed, sumFiles abandons its work.
func sumFiles(done <-chan struct{}, root string) (<-chan result, <-chan error) {
	// For each regular file, start a goroutine that sums the file and sends
	// the result on c.  Send the result of the walk on errc.
	c := make(chan result)
	errc := make(chan error, 1)
	go func() { // HL
		var wg sync.WaitGroup
		err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.Mode().IsRegular() {
				return nil
			}
			wg.Add(1)
			go func() { // HL
				data, err := ioutil.ReadFile(path)
				select {
				case c <- result{path, md5.Sum(data), err}: // HL
				case <-done: // HL
				}
				wg.Done()
			}()
			// Abort the walk if done is closed.
			select {
			case <-done: // HL
				return errors.New("walk canceled")
			default:
				return nil
			}
		})
		// Walk has returned, so all calls to wg.Add are done.  Start a
		// goroutine to close c once all the sends are done.
		go func() { // HL
			wg.Wait()
			close(c) // HL
		}()
		// No select needed here, since errc is buffered.
		errc <- err // HL
	}()
	return c, errc
}

// MD5All reads all the files in the file tree rooted at root and returns a map
// from file path to the MD5 sum of the file's contents.  If the directory walk
// fails or any read operation fails, MD5All returns an error.  In that case,
// MD5All does not wait for inflight read operations to complete.
func MD5All(root string) (map[string][md5.Size]byte, error) {
	// MD5All closes the done channel when it returns; it may do so before
	// receiving all the values from c and errc.
	done := make(chan struct{}) // HLdone
	defer close(done)           // HLdone

	c, errc := sumFiles(done, root) // HLdone

	m := make(map[string][md5.Size]byte)
	for r := range c { // HLrange
		if r.err != nil {
			return nil, r.err
		}
		m[r.path] = r.sum
	}
	if err := <-errc; err != nil {
		return nil, err
	}
	return m, nil
}

func WriteToFile(filename string, data string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.WriteString(file, data)
	if err != nil {
		return err
	}
	return file.Sync()
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

var arguments docopt.Opts

func main() {

	var usage = `Usage: md5folder [DIR] [-hms]

	Arguments:
	  DIR        dir to process
	
	Options:
	  -h      help
	  -m      md5
	  -s      stat

	`

	arguments, _ = docopt.ParseDoc(usage)
	//fmt.Println(arguments)
	if arguments["-m"].(bool) {
		fmt.Println("calc md5")
		calcMd5()
	} else if arguments["-s"].(bool) {
		fmt.Println("calc stats")
	} else {
		fmt.Println("no args provided. use -h for help")
	}

}

func calcMd5() {
	md5ListFile := os.Args[1] + string(filepath.Separator) + ".md5list"

	if fileExists(md5ListFile) {
		fmt.Println(".md5list files exists. This directory already processed. Exiting.")
		return
	}

	//fmt.Println("length", len(os.Args))
	if len(os.Args) <= 1 {
		fmt.Println("Requires path as paramater")
		return
	}

	var buffer bytes.Buffer

	buffer.WriteString(fmt.Sprintf("md5 hash of directory contents v1.0\n"))

	// Calculate the MD5 sum of all files under the specified directory,
	// then print the results sorted by path name.
	m, err := MD5All(os.Args[1])
	if err != nil {
		fmt.Println(err)
		return
	}
	var paths []string
	for path := range m {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		//ignore hidden files
		if string(path[0]) != "." {
			line := fmt.Sprintf("%x  %s\n", m[path], path)
			fmt.Printf(line)
			buffer.WriteString(line)
		}
	}

	err = WriteToFile(md5ListFile, buffer.String())
	if err != nil {
		fmt.Println(err)
	}

}
