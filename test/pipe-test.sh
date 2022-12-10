#!/bin/bash

cd `dirname $0`

while read -r LINE; do
    echo $LINE
    sleep 1
done < ./data/xml/formatted.xml
