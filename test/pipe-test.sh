#!/bin/bash

FILE=$1

while read -r LINE; do
    echo $LINE
    sleep 1
done < $FILE
