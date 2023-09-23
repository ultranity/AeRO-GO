import socketserver

#创建一个socketserver类继承socketserver模块下的BaseRequestHandler类
class MyServer(socketserver.BaseRequestHandler):
    def handle(self):
        # 第二步：服务器端阻塞，等待客户端连接
        #重写父类中的handle方法，主要实现服务端的逻辑代码，，不用写bind() listen() accept()
        while True:
            conn = self.request
            addr = self.client_address
            print(conn)  #<socket.socket fd=864, family=AddressFamily.AF_INET, type=SocketKind.SOCK_STREAM, proto=0, laddr=('127.0.0.1', 9999), raddr=('127.0.0.1', 50565)>
            print(addr)  #('127.0.0.1', 50565)
            # 第三步：服务器端与客户端进行通信
            #接收到来自客户端的数据
            recv_data = str(conn.recv(1024),encoding = 'utf8')
            print(recv_data)
            #如果客户端发送的是‘bye’就断开连接
            if recv_data == 'bye':
                break
            # 服务器端回复数据给客户端
            send_data =  bytes(input('请输入回复消息：'),encoding = 'utf8')
            conn.sendall(send_data)
        conn.close()
if __name__ == '__main__':
    # 第一步
    #实例化server对象，传入本机ip，以及监听的端口号，还有新建的继承socketserver模块下的BaseRequestHandler类
    server = socketserver.ThreadingTCPServer(('127.0.0.1',9999),MyServer)
    #激活服务端
    server.serve_forever()