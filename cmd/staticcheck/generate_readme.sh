#!/bin/sh

build() {
    echo "|Check|Description|"
    echo "|---|---|"
    for cat in docs/categories/*; do
        catname=$(basename "$cat")
        catdesc=$(cat "$cat")
        echo "|**$catname???**|**$catdesc**|"

        for check in docs/checks/"$catname"*; do
            checkname=$(basename "$check")
            checktitle=$(head -1 "$check")
            checkdesc=$(tail -n +3 "$check")

            if [ -n "$checkdesc" ]; then
                echo "|[$checkname](#$checkname)|$checktitle|"
            else
                echo "|$checkname|$checktitle|"
            fi
        done
        echo "|||"
    done

    echo

    for check in docs/checks/*; do
        checkname=$(basename "$check")
        checktitle=$(head -1 "$check")
        checkdesc=$(tail -n +3 "$check")

        if [ -n "$checkdesc" ]; then
            echo "### <a id=\"$checkname\">$checkname – $checktitle"
            echo
            echo "$checkdesc"
        fi
    done
}

output=$(build)

while IFS= read -r line; do
    if [ "$line" = "[CHECKS PLACEHOLDER]" ]; then
        echo "$output"
    else
        echo "$line"
    fi
done < README.md.template
