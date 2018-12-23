#!/bin/bash

START_PROCESS_LOCK=/tmp/start_mt.lock

#get current script run path
function getCurrentPath()
{
    if [ "`dirname $0`" = "" ] || [ "`dirname $0`" = "." ]; then
        currentPath="`pwd`"
    else
        cd `dirname $0`
        currentPath="`pwd`"
        cd - > /dev/null 2>&1
    fi
}

#set app base env
function setBaseDir()
{
    cd $currentPath/..
    APP_BASE_DIR="`pwd`"
    export APP_BASE_DIR
}

#start process
function start_process()
{
    status && die "proc has started."

    nohup ./nicemqtt >/var/log/nicemqtt/sys_error.log 2>&1 &

    sleep 3
}

function main()
{
    echo "starting..."

    getCurrentPath

    setBaseDir

    cd $currentPath
    . ./util.sh

    initLog

#    chkUser

    lockWrapCall "${START_PROCESS_LOCK}" start_process
    
    status || die "start nicemqtt failed."
    logger "start nicemqtt success."
}

main $*
exit $?
