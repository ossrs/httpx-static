#!/usr/bin/env bash

cwd=`dirname $0` && cd $cwd
if [[ ! -d apilb ]]; then
    if [[ -d ../apilb ]]; then
        cd ..
    else
        echo "no apilb" && exit 1
    fi
fi
echo "current dir: `pwd`"

(echo "build apilb" && cd apilb/ && go build . && echo "apilb ok") &&
(echo "build httplb" && cd httplb/ && go build . && echo "httlb ok") &&
(echo "build rtmplb" && cd rtmplb/ && go build . && echo "rtmplb ok") &&
(echo "build shell" && cd shell/ && go build . && echo "shell ok") && 
(echo "create objs" && cd shell/ && mkdir -p objs && cd objs && echo "objs ok") &&
(echo "link srs" && cd shell/objs/ && ln -sf ~/srs/objs/srs && echo "srs ok") &&
echo "build success, you can:" &&
echo "      cd `pwd`/shell && ./shell  -c ../conf/shell.json" &&
exit 0

ret=$? && echo "build failed, code=$ret" && 
exit $ret


