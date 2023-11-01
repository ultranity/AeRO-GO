# 本地回环测试
初代
-R
[ ID] Interval           Transfer     Bandwidth       Retr
[  4]   0.00-10.00  sec  6.85 GBytes  5.88 Gbits/sec    0             sender
[  4]   0.00-10.00  sec  6.84 GBytes  5.87 Gbits/sec                  receiver

copy make pool
[ ID] Interval           Transfer     Bandwidth       Retr
[  4]   0.00-10.00  sec  6.85 GBytes  5.89 Gbits/sec    0             sender
[  4]   0.00-10.00  sec  6.85 GBytes  5.88 Gbits/sec                  receiver
-R 
[  4]   0.00-10.00  sec  6.90 GBytes  5.96 Gbits/sec    0             sender
[  4]   0.00-10.00  sec  6.89 GBytes  5.95 Gbits/sec                  receiver

换用 go-yamux v4
[ ID] Interval           Transfer     Bandwidth       Retr
[  4]   0.00-10.00  sec  14.0 GBytes  12.0 Gbits/sec    0             sender
[  4]   0.00-10.00  sec  14.0 GBytes  12.0 Gbits/sec                  receiver
-R
[  4]   0.00-10.00  sec  14.4 GBytes  12.4 Gbits/sec    5             sender
[  4]   0.00-10.00  sec  14.4 GBytes  12.4 Gbits/sec                  receiver
换用 go-yamux v4 但无 copy make pool
[ ID] Interval           Transfer     Bandwidth       Retr
[  4]   0.00-10.00  sec  13.2 GBytes  11.4 Gbits/sec    0             sender
[  4]   0.00-10.00  sec  13.2 GBytes  11.4 Gbits/sec                  receiver
-R
[  4]   0.00-10.00  sec  14.4 GBytes  12.4 Gbits/sec    0             sender
[  4]   0.00-10.00  sec  14.4 GBytes  12.4 Gbits/sec                  receiver

使用 32k buf 需cpu充足
[ ID] Interval           Transfer     Bandwidth       Retr
[  4]   0.00-10.00  sec  18.3 GBytes  15.7 Gbits/sec    0             sender
[  4]   0.00-10.00  sec  18.3 GBytes  15.7 Gbits/sec                  receiver



## rathole
docker run -p 2333:2333 -p 5555:5555 -it --rm -v "/root/config.toml:/app/config.toml" rapiz1/rathole --server /app/config.toml
docker run -it --network host --rm -v "/root/config.toml:/app/config.toml" rapiz1/rathole --client /app/config.toml

[ ID] Interval           Transfer     Bandwidth       Retr
[  4]   0.00-10.00  sec  5.45 GBytes  4.68 Gbits/sec    9             sender
[  4]   0.00-10.00  sec  5.43 GBytes  4.67 Gbits/sec                  receiver
-R
[  4]   0.00-10.00  sec  5.30 GBytes  4.55 Gbits/sec    0             sender
[  4]   0.00-10.00  sec  5.28 GBytes  4.54 Gbits/sec                  receiver

# real world 内蒙to深圳
## aeroc 
[ ID] Interval           Transfer     Bandwidth       Retr
[  4]   0.00-10.00  sec  53.8 MBytes  45.1 Mbits/sec    0             sender
[  4]   0.00-10.00  sec  44.9 MBytes  37.6 Mbits/sec                  receiver

第二轮
[ ID] Interval           Transfer     Bandwidth       Retr
[  4]   0.00-10.00  sec   143 MBytes   120 Mbits/sec    0             sender
[  4]   0.00-10.00  sec   134 MBytes   112 Mbits/sec                  receiver
-R
[ ID] Interval           Transfer     Bandwidth       Retr
[  4]   0.00-10.00  sec   138 MBytes   115 Mbits/sec   17             sender
[  4]   0.00-10.00  sec   128 MBytes   108 Mbits/sec                  receiver

## frp
[ ID] Interval           Transfer     Bandwidth       Retr
[  4]   0.00-10.00  sec  29.6 MBytes  24.8 Mbits/sec    0             sender
[  4]   0.00-10.00  sec  20.8 MBytes  17.5 Mbits/sec                  receiver
-R
[  4]   0.00-10.00  sec  52.5 MBytes  44.0 Mbits/sec   28             sender
[  4]   0.00-10.00  sec  43.7 MBytes  36.7 Mbits/sec                  receiver
第二轮
[ ID] Interval           Transfer     Bandwidth       Retr
[  4]   0.00-10.00  sec  38.5 MBytes  32.3 Mbits/sec    0             sender
[  4]   0.00-10.00  sec  29.8 MBytes  25.0 Mbits/sec                  receiver


# real world 宿迁to深圳
## aeroc
[ ID] Interval           Transfer     Bandwidth       Retr
[  4]   0.00-10.00  sec   138 MBytes   115 Mbits/sec    0             sender
[  4]   0.00-10.00  sec   128 MBytes   107 Mbits/sec                  receiver
-R
[ ID] Interval           Transfer     Bandwidth       Retr
[  4]   0.00-10.00  sec   138 MBytes   115 Mbits/sec    0             sender
[  4]   0.00-10.00  sec   128 MBytes   107 Mbits/sec                  receiver

## ssh reverse
[ ID] Interval           Transfer     Bandwidth       Retr
[  4]   0.00-10.00  sec   145 MBytes   121 Mbits/sec    0             sender
[  4]   0.00-10.00  sec   136 MBytes   114 Mbits/sec                  receiver
-R
[  4]   0.00-10.00  sec   138 MBytes   115 Mbits/sec   51             sender
[  4]   0.00-10.00  sec   129 MBytes   108 Mbits/sec                  receiver