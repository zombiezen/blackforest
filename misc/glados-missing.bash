#!/bin/bash

for name in $(glados ls)
do
    glados path "$name" > /dev/null
    if [ $? -ne 0 ]
    then
        echo "$name"
    fi
done
