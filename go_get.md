## 1 go get 使用私有仓库
	
	$ git config --global url."git@github.com:" .insteadOf "https://github.com/"
	$ cat ~/.gitconfig
	[url "git@github.com:"]
	     insteadOf = https://github.com/
	     
	     
## 2 go get 使用fork项目

### 2.1 add remote
                                       fork
	http://github.com/awsome-org/tool  ------>  http://github.com/awesome-you/tool
	
	
	$ go get http://github.com/awesome-org/tool
	$ git remote add awesome-you-fork http://github.com/awesome-you/tool
	
	$ git pull --rebase awesome-you-fork
	$ git push awesome-you-fork
	
### 2.2 cheat "go get"

	cd $GOPATH
	mkdir -p {src,bin,pkg}
	mkdir -p src/github.com/awesome-org/
	cd src/github.com/awesome-org/
	git clone git@github.com:awesome-you/tool.git # OR: git clone https://github.com/awesome-you/tool.git
	cd tool/
	go get ./...
	
	
link 

* [Forking Golang repositories on GitHub and managing the import path](http://code.openark.org/blog/development/forking-golang-repositories-on-github-and-managing-the-import-path)
* [GitHub and Go: forking, pull requests, and go-getting](http://blog.campoy.cat/2014/03/github-and-go-forking-pull-requests-and.html)