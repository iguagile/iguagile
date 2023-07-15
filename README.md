# iguagile

Network synchronization of objects in 3d space with many people and low latency.

This project is experimental and not ready for production

## Quick Start

```bash
git clone git@github.com:iguagile/iguagile.git
cd iguagile
docker-compose up
curl http://localhost:8080/api/v1/rooms -X POST -d '{"application_name": "example", "version": "0.1.0", "password": "IiHqswslP2Yr3b3P", "max_user": 4, "information": {}}'
# response
# {"success":true,"result":{"room_id":65536,"require_password":false,"max_user":0,"connected_user":0,"server":{"server":"192.168.10.5","port":10000},"token":"BHB2dVhpT1GcP4IKN9iLJw==","information":null},"error":""}
# connect to 192.168.10.5:10000
```
