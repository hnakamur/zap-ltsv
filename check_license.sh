#!/bin/bash

text=`head -1 LICENSE.txt`
text2=`head -2 LICENSE.txt | tail -1`

ERROR_COUNT=0
while read file
do
    head -1 ${file} | grep -q "${text}"
    if [ $? -ne 0 ]; then
        head -1 ${file} | grep -q "${text2}"
        if [ $? -ne 0 ]; then
            echo "$file is missing license header."
            (( ERROR_COUNT++ ))
	fi
    fi
done < <(git ls-files "*\.go")

exit $ERROR_COUNT
