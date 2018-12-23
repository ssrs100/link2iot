#!/bin/bash

#!/bin/bash


#get current script run path
function getCurrentPath()
{
    if [ "` dirname $0 `" = "" ] || [ " ` dirname $0 ` " = "." ]; then
        currentPath="`pwd`"
    else
        cd `dirname $0`
        currentPath="`pwd`"
        cd - > /dev/null 2>&1
    fi
}

function main()
{
    echo "stopping..."

    getCurrentPath
    cd $currentPath

    . ./util.sh
    initLog

#    chkUser

    status || die "proc has stopped."
    #kill proc graceful first
    local pid=$(ps -ww -eo pid,cmd | grep -w "${MODULE_NAME}$" | grep -vwE "grep|vi|vim|tail|cat" | awk '{print $1}' | head -1)
    kill $pid

    sleep 1    
    # check and kill forced
    status
    if [ $? -eq 0 ]; then
        kill -9 ${pid} > /dev/null 2>&1
        logger "stop ${MODULE_NAME} forced."
    fi
    echo "stop ${MODULE_NAME} success"
}

main $*
exit $?

