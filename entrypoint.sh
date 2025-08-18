#!/bin/bash

find $1 -name '*.yaml' | xargs /secret-tunnel/app
