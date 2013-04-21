#!/bin/bash

for name in $(blackforest ls)
do
    blackforest path "$name" > /dev/null
    if [ $? -ne 0 ]
    then
        echo "$name"
    fi
done
