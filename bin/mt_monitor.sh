#!/bin/bash


LOOP_COUNTER=3

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

#set app base env
function setBaseDir()
{
    cd $currentPath/..
    APP_BASE_DIR="`pwd`"
    export APP_BASE_DIR
}

# for status
function innerStatus()
{
    if status >/dev/null ; then
        logger "normal"
    else
        logger "abnormal"
        return 2
    fi
}

# for start
function start()
{
    if status >/dev/null ; then
        logger "process is running, no need to start"
        return 2
    fi

    # start process
    sh ./start_mt.sh
    if [ $? -eq 0 ] ; then
        logger "start success"
        return 0
    else
        die "start fail"
    fi
}

# for stop
function stop()
{
    if status >/dev/null ; then
        logger "process is running, try to stop it"
    else
        logger "process is not running, no need to stop"
        return 2
    fi

    # stop process
    sh ./stop_mt.sh
    if [ $? -eq 0 ] ; then
        logger "stop success"
        return 0
    else
        die "stop fail"
    fi
}

# for restart
function restart()
{
    stop
    start
}

# for check
function check()
{
    # store http result in tmp 
    logger "ready for check start"
    local CHECK_TMP=$currentPath/../conf/CHECK.TMP
    # rm check.tmp
    if [ -f  "${CHECK_TMP}" ]; then
        logger "remove CHECK.TMP"
        rm ${CHECK_TMP}
    fi
    
    # read ip and port
    getPort
    getIp 
    # check ip and port
    if [[ -z $IP || -z $PORT ]]; then
        logger "ip or port does not found"
        logger "check: abnormal"
        return 2
    fi
    
    # check status start
    # The current number of cycles
    local CURRENT_NUMBER=0
    
    for((; CURRENT_NUMBER < ${LOOP_COUNTER}; CURRENT_NUMBER++));
    do
        # if status is ok
        status > /dev/null 2>&1
        if [ $? -eq 0 ] ;then
            logger "Result:check success, CURRENT_NUMBER is ${CURRENT_NUMBER}"
            return 0
        else
            logger "Result:check failed, CURRENT_NUMBER is ${CURRENT_NUMBER}. it will start it."
            sh ./start_mt.sh
            
            local mycommand="/usr/bin/curl" 
            if [ -f $mycommand ] && [ -x $mycommand ]; then
                logger "it will curl it."
                curl -X GET -I -k --connect-timeout 3 -m 3 "https://${IP}:${PORT}/v1/heart" > ${CHECK_TMP} 2>/dev/null
                # status 200 is ok
                local status=$(sed -n '/HTTP/p' ${CHECK_TMP} | awk -F ' ' '{print $2}')
                
                # rm check.tmp
                if [ -f  "${CHECK_TMP}" ]; then
                    rm ${CHECK_TMP}
                fi
                
                # if status is null
                if [ -z ${status} ]; then
                    logger "code is 500, status check: abnormal, try to stop it."                
                    sh ./stop_mt.sh
                elif [ ${status} == '200' ]; then
                    logger "code is ${status}, status check: normal"
                    return 0
                else
                    logger "code is ${status}, status check: abnormal, try to stop it."
                    sh ./stop_mt.sh
                fi
            else 
                logger "just to sleep 2 seconds."
                sleep 5
            fi
        fi
    done
    
    return 2
}


function main()
{
    getCurrentPath

    setBaseDir

    cd $currentPath
    . ./util.sh

    initLog

#    chkUser

    ACTION=$1
    [ -z $ACTION ] && die $"Usage: $0 {start|stop|status|restart|check}"
     
    case "$ACTION" in
        start)
        start
        ;;
        stop)
        stop
        ;;
        status)
        innerStatus
        ;;
        restart)
        restart
        ;;
        check)
        check
        ;;
        *)
        die $"Usage: $0 {start|stop|status|restart|check}"
    esac
    return $?
}

main $*
exit $?
