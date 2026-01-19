# COps Daemon

- Global Config를 Watch하고 변경을 감지해서 Watch가 필요한 Claude Code Project를 수집

## Global Config로부터 Watch할 Project 세팅하기
1. `~/.cops/config.json` 파일을 Watch
2. Config 파일에서 Project Directory 확인
3. Project Directory가 Git Project인지 확인
   1. Git Project인 경우? Worktree 리스트 모두 Watch
   2. Git Project가 아닌 경우? 해당 디렉토리만 Watch

## Log 변경 발생?
1. 변경점 Parsing해서 API 서버에 보내주기. DEBUG 옵션이 켜져있으면 매번 즉시 전송. 아니라면 특정 주기마다 Flush 하는 방식으로 구현하기
