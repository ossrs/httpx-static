#!/bin/bash

if [[ $# -lt 1 ]]; then
    echo "Usage $0: <url>"
    echo "      url the hls+ url to download."
    echo "For example:"
    echo "      $0 http://localhost:8080/live/livestream.m3u8"
    echo "      $0 http://localhost:8080/live/livestream.m3u8?shp_identify=variant"
    echo "      $0 http://localhost:8080/live/livestream.m3u8?shp_identify=302"
    exit 1
fi

url=$1
variant=no && echo $url| grep "shp_identify=variant" >/dev/null 2>&1 && variant=yes
echo "Download the $url, variant:$variant"

stream=`echo $url| awk -F '?' '{print $1}'` && dir=`dirname $stream` && stream=`basename $stream`
echo "Remote dir is $dir, stream is $stream"

key=""
m3u8=""
if [[ $variant == yes ]]; then
    master="master-${stream}"
    echo "Discovery the key in variant $master"
    curl "$url" -o $master -s
    if [[ $? -ne 0 ]]; then 
        echo "Discovery variant HLS failed."; exit 1; 
    fi
else
    master="master-${stream%.*}.txt"
    echo "Discovery the key in 302 in $master"
    curl "$url" -v 2>&1 |grep Location|awk '{print $3}' > $master &&
    dos2unix $master
    if [[ $? -ne 0 ]]; then 
        echo "Discovery 302 HLS failed."; exit 1; 
    fi
fi
m3u8=`cat $master |grep m3u8|awk -F '?' '{print $1}'` &&
key=`cat $master |grep m3u8|awk -F '?' '{print $2}'`
if [[ $? -ne 0 ]]; then 
    echo "Failed."; exit 1; 
fi
echo "Key is $key";
echo "M3u8 is $m3u8";

for ((;;)); do
    m3u8_url="${m3u8}?${key}"
    curl "${m3u8_url}" -o $stream -s &&
    files=`cat $stream |grep ts`
    if [[ $? -ne 0 ]]; then
        echo "Download $m3u8_url failed."
        exit 1
    fi
    
    for file in $files; do 
        filename=`echo $file| awk -F '.ts' '{print $1}'`.ts && 
        if [[ ! -f $filename ]]; then 
            tsfile="${dir}/${file}"
            echo "Download $tsfile to $filename" && 
            curl "$tsfile" -o ${filename} -s
        fi 
        if [[ $? -ne 0 ]]; then
            echo "Download $filename failed."
            exit 1
        fi
    done
    sleep 3 
done
