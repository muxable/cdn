#!/bin/bash

docker run -v $PWD:/defs namely/protoc-all -f cdn.proto -l go