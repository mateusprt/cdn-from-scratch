# CDN From Scratch

This project aims to build a simple CDN to better understand how it works. The implementation provides a simplified representation of a real-world CDN architecture, consisting of the following components:

- **Routing System**: determines the edge server responsible for content delivery. There is a simplification here: in a real-world CDN, the routing system determines the optimal path based on factors such as network proximity and server availability. In this project, the routing system is implemented using NGINX as a load balancer, which distributes requests among the Edge Servers using a consistent hashing strategy;
  
- **Origin Server**: stores the original content;
  
- **Scrubber Server**: filters malicious requests and helps protect against attacks such as DDoS. In this project, this component only implements rate limiting.
  
- **Proxy Edge Server**: caches and serves content to clients;

- **Logging and Monitoring Tools**: Prometheus, Grafana and Jaegger;

