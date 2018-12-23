#/bin/bash

OK=0
COMMON_ERROR=1

MODULE_NAME=nicemqtt

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


function die()
{
    echo "$*"
    exit ${COMMON_ERROR}
}

#clean last build result
function clean()
{
    rm bin/${MODULE_NAME} 2>/dev/null
    rm bin/${MODULE_NAME}-* 2>/dev/null
}

#check for build
function check()
{
    go version > /dev/null 2>&1
    if [ $? -ne 0 ]; then
        echo "GO should be installed first."
        return $COMMON_ERROR
    fi
    return $OK
}

function build()
{
    if [ -z $1 ]; then
        echo "build need a target param"
        return $COMMON_ERROR
    fi

    if [[ -z $2 || "$2" == "release" ]]; then
        go build -o bin/$1 src/nicemqtt.go
    elif [[ "$2" == "debug" ]];then
        go build -o bin/$1 -gcflags="-N -l" src/nicemqtt.go
    else
        echo "Usage:$0 [debug|release]"
        echo "build failed!"
        return $COMMON_ERROR
    fi

    if [ $? -eq 0 ] ; then
        echo "build success!"
    else
        echo "build failed!"
        return $COMMON_ERROR
    fi
    return $OK
}

function main()
{

    version=$1
    [ -z ${version} ] && die $"Usage: $0 version [release|debug]"

    echo "build starting..."

    getCurrentPath

    export GOPATH=${currentPath}

    cd ${currentPath}

    clean

    check || exit $?

    target=${MODULE_NAME}-${version}
    buildtype=$2
    build ${target} ${buildtype}
    cd bin/
    ln -s ${target} ${MODULE_NAME}
    exit $?
}

main $*
exit $?