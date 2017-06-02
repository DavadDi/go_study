#/bin/bash
# modify backlog, meanwhile change listen function to 1024, default 128
echo "1024" > /proc/sys/net/core/somaxconn
sysctl -w net.core.somaxconn=1024
 
# more ports for testing
sysctl -w net.ipv4.ip_local_port_range="1025 65535"  # 1025 - 6535, about 50000 port avaiable

# tcp read buffer, min, default, maximum
sysctl -w net.ipv4.tcp_rmem="2048 2048 16777216"      # adjust read buffer -> 2k

# tcp write buffer, min, default, maximum
sysctl -w net.ipv4.tcp_wmem="2048 2048 16777216"      # djust write buffer -> 2k, so every socket take about 4k bytes, tested 3.75k.

# tcp  buffer, min, default, maximum, uint = page,page = 4096 byte, we can change the value based on mem
sysctl -w net.ipv4.tcp_mem="1250000 1500000 1750000"  # min 5G, default 6G max 7G, 7G/4K about 175w connections

# tcp
sysctl -w net.ipv4.tcp_max_orphans="200000"          # as large as possible, consider physical mem.

echo 9999999 | tee /proc/sys/fs/nr_open
echo 9999999 | tee /proc/sys/fs/file-max

ulimit -n 9999999

#ulimit -HSn 9999999, setting soft link and hard link

# edit /etc/security/limits.conf, add line, for all user
#* - nofile 9999999
