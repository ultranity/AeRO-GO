import socket
import time

def test_latency(ip, port, num_tests=10):
    latencies = []
    for _ in range(num_tests):
        client_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        client_socket.connect((ip, port))
        start_time = time.perf_counter()
        print("start_time",start_time)
        client_socket.sendall(b"Test data")
        client_socket.recv(1024)
        end_time = time.perf_counter()
        print("end time",end_time)
        latency = end_time - start_time
        latencies.append(latency)
        client_socket.close()
    average_latency = sum(latencies)/num_tests
    print(f"Average latency: {average_latency} seconds")

test_latency("localhost", 7000)