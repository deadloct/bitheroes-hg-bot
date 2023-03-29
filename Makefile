ifneq ("$(wildcard .env)", "")
	include .env
endif

NAME := bitheroes-hg-bot
LOCAL_PATH := bin

clean:
	rm -rf $(LOCAL_PATH)

test:
	go test ./... -test.v -race

bench:
	go test -bench=. -benchtime=10s -benchmem ./...

build: clean
	go build -o $(LOCAL_PATH)/$(NAME)
	cp -R data $(LOCAL_PATH)/

run: build
	$(LOCAL_PATH)/$(NAME)

ssh_build: clean
	GOOS=linux GOARCH=arm64 GOARM=5 go build -o $(LOCAL_PATH)/$(NAME)
	cp -R data $(LOCAL_PATH)

ssh_deploy: ssh_build
	rsync -avz $(LOCAL_PATH)/ $(SSH_HOST):$(SSH_DIR)
	-ssh $(SSH_HOST) "killall $(NAME)"

docker_build:
	docker build -t $(IMAGE_URL) .

docker_run: docker_build
	docker run -e "BITHEROES_HG_BOT_AUTH_TOKEN=$(BITHEROES_HG_BOT_AUTH_TOKEN)" $(IMAGE_URL)

docker_push: docker_build
	docker push $(IMAGE_URL)

k8s_deploy:
	kubectl apply -f k8s
