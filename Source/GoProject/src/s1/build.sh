#!/bin/bash
rm -rf s1
xgo run 'cd JuusanKoubou/Source/GoProject/src/s1; go build s1.go && upx -9 --lzma s1'
scp -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no s1 root@23.102.224.63:~/

