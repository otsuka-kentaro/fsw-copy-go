# add library
```shell script
$ docker-compose run --rm -v $(pwd):/app builder go get [library_name]
```

# upload image
```shell script
$ docker build -t okentaro/fsw-copy-go .
$ docker login 
$ docker push okentaro/fsw-copy-go:latest
```
