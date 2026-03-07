CONTAINERS = postgres encargalo-redis redis-interface

up:
	sg docker -c "docker start $(CONTAINERS)"

down:
	sg docker -c "docker stop $(CONTAINERS)"

status:
	sg docker -c "docker ps -a --filter name=postgres --filter name=encargalo-redis --filter name=redis-interface"
