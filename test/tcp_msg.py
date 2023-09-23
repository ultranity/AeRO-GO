import socket

#创建一个socket对象，指定要连接的目标服务器 ip及端口号
# 第一步
s =  socket.socket()
# 第二步
s.connect(('127.0.0.1',9999))
while True:
    #连接成功后向服务器端发送数据 
    send_data = input('请输入需要发送的内容')
    # 第三步
    s.sendall(bytes(send_data,encoding = 'utf8'))
    if send_data=='bye':
        break
    #客户端接收来自服务器端发送的数据
    recv_data =  str(s.recv(1024),encoding='utf8')
    print(recv_data)
s.close()