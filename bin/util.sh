#!/bin/bash

SERVICE_USER=nicemqtt
SERVICE_GROUP=nicemqtt
PRODUCT_NAME=nicemqtt
MODULE_NAME=nicemqtt

PRODUCT_PATH=/opt/${SERVICE_USER}/${MODULE_NAME}
PRODUCT_BIN_DIR=${PRODUCT_PATH}/bin
PRODUCT_CONF_DIR=${PRODUCT_PATH}/conf

#对应缺省：目录700 文件600权限
umask 0077

RETURN_CODE_SUCCESS=0
RETURN_CODE_ERROR=1

LOGMAXSIZE=5120
BASE_LOGGER_PATH=/var/log/${SERVICE_USER}


chkUser()
{
    logger_without_echo "check current user"
    local curUser=$(/usr/bin/whoami | /usr/bin/awk '{print $SERVICE_USER}')
    if [ "$curUser" = "$SERVICE_USER" ]; then
       logger_without_echo "check current user success"
       return 0
    else
       die "${MODULE_NAME} can only run by ${SERVICE_USER}"
    fi
}


function chkRootUser()
{
    logger_without_echo "check root user"
    local curUser=$(/usr/bin/whoami | /usr/bin/awk '{print $1}')
    if [ "$curUser" = "root" ]; then
       logger_without_echo "check root user success"
       return 0
    else
       die "This operation could be handled only by root."
    fi
}


initLog()
{
    LOGGER_PATH=${BASE_LOGGER_PATH}/${MODULE_NAME}
    LOGGER_FILE=${LOGGER_PATH}/${PRODUCT_NAME}-sh.log
    if [ -e "$LOGGER_PATH" ]; then
        return 0
    else
        mkdir -p ${LOGGER_PATH}
        echo "init log dir success."
    fi
}

status()
{
    # simple check for process
    local pid=$(ps -ww -eo pid,cmd | grep -w "${MODULE_NAME}$" | grep -vwE "grep|vi|vim|tail|cat" | awk '{print $1}' | head -1)
    if [ $pid ]; then
        return $RETURN_CODE_SUCCESS
    else
        return $RETURN_CODE_ERROR
    fi
}


logger_without_echo()
{
    local logsize=0
    if [ -e "$LOGGER_FILE" ]; then
        logsize=`ls -lk ${LOGGER_FILE} | awk -F " " '{print $5}'`
    else
        touch ${LOGGER_FILE}
#        chown ${SERVICE_USER}: ${LOGGER_FILE}
        chmod 600 ${LOGGER_FILE}
    fi

    if [ "$logsize" -gt "$LOGMAXSIZE" ]; then
        # 每次删除10000行，约300K
        sed -i '1,10000d' "$LOGGER_FILE"
    fi
    echo "[` date -d today +\"%Y-%m-%d %H:%M:%S\"`,000] $*" >>"$LOGGER_FILE"

}

logger()
{
    logger_without_echo $*
    echo "$*"
}


die()
{
    logger "$*"
    exit ${RETURN_CODE_ERROR}
}

#get process port
getPort()
{
    #PORT="`sed '/\"port\"/!d;s/[^0-9]//g' ../conf/nicemqtt.json`"
    PORT=8080
}

#get process ip
getIp()
{
    IP="`sed '/\"host\"/!d;s/[^0-9.]//g' ../conf/nicemqtt.json`"
}


ERR_GET_LOCK=101
ERR_PARAMETERS=102
#######################################################################
# shell文件锁封装函数，
# 参数1：文件锁路径，
# 参数2，需要上锁的实际执行动作，
# 参数3~n，传给实际动作的所有参数
lockWrapCall()
{
    local lockFile=$1
    local action=$2

    [ -n "$lockFile" -a -n "$action" ] || return $ERR_PARAMETERS

    local dirName=$(dirname $lockFile)
    [ -d "$dirName" ] || return $ERR_PARAMETERS

    shift 2

    # 定义信号捕捉流程，异常停止进程时清除文件锁
    trap 'rm -f $lockFile; logger "trap a stop singal"' 1 2 3 15

    ####################################
    ## 文件锁，只允许一个进程执行
    ####################################
    {
        flock -no 100
        if [ $? -eq 1 ]; then
            local lockPid=$(cat $lockFile)
            lockPid=$(echo $lockPid)
            if [ -z "$lockPid" ]; then
                logger "can't get lock file:$lockFile, lockPid is empty, no need to run $action"
                return $ERR_GET_LOCK
            else
                lockPid=$(echo $lockPid)
                local openPids=$(lsof -Fp $lockFile)
                if echo "$openPids" | grep "^p${lockPid}$" > /dev/null; then
                    logger "can't get lock file:$lockFile, lockPid:$lockPid is running, no need to run $action"
                    return $ERR_GET_LOCK
                fi
            fi
            logger "success get lock file:$lockFile, lockPid:$lockPid is not running"
        fi
        echo $$ > $lockFile

        $action "$@"
        local ret=$?

        # 删除文件锁，使得上述动作参数的子进程不再持有锁
        rm -f $lockFile
        return $ret
    } 100<>$lockFile

    # 恢复为默认信号处理
    trap '-' 1 2 3 15
}
