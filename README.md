# md5folder
md5 hash a directory



A golang equivalent of the following
```
find . -type f -exec md5sum {} +
```

Create a file named .md5list with the md5 hash of each file in the directory specified.
Hidden files are skipped.


print filename and file size
```
find . -ls | awk '{print $11, $7}' | sort
```

Build a Power PC binary
env GOOS=linux GOARCH=ppc64le go build main.go

