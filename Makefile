.PHONY: build clean install lint

all: build

build:
	go build -v -o ./static .

clean:
	rm ./static

install:
	go install

lint:
	$(GOPATH)/bin/golint .
