import socketserver
import time
class MyServer(socketserver.BaseRequestHandler):
    def handle(self):
        while True:
            data = self.request[0].strip()
            socket = self.request[1]
            print('客户端IP:',self.client_address[0])    # 192.168.141.1
            socket.sendto(data, self.client_address)
            time.sleep(2)
        conn.close()
if __name__ == '__main__':
    server = socketserver.ThreadingUDPServer(('0.0.0.0',9999),MyServer)
    print('servering……')
    server.serve_forever()