# C-Ops API 서버 가이드

API 서버는 C-Ops 시스템의 "두뇌"이자 "기억 저장소"입니다. 데몬으로부터 데이터를 받아 영구 저장하고, 대시보드의 요청에 따라 데이터를 가공하여 제공합니다.

## 1. 역할 및 동작 원리

### 역할
- **Collector (Ingestion)**: 데몬들로부터 전송되는 로그 데이터를 고속으로 수신하고 검증합니다.
- **Storage Gateway**: 수신된 데이터를 MongoDB에 효율적으로 저장합니다.
- **Query Engine (Dashboard)**: 웹 대시보드의 요청에 따라 프로젝트별, 세션별 데이터를 조회하고 집계(Aggregation)합니다.

### 동작 원리
Go 언어 기반의 `Fiber`(HTTP) 및 `ConnectRPC`(gRPC) 프레임워크를 사용하여 구현되었습니다. 단일 바이너리로 실행되지만 내부적으로는 **Collector**와 **Dashboard**라는 두 가지 논리적 서비스로 나뉩니다.

## 2. 서비스 아키텍처

데몬과 마찬가지로 `fx` 프레임워크를 사용한 DI(의존성 주입) 구조를 가집니다.

### 컴포넌트
- **CollectorService**: `gRPC`. 데몬이 바라보는 엔드포인트입니다. `SendRecords` RPC를 처리합니다.
- **DashboardService**: `gRPC` (Connect-Web). 웹 프론트엔드가 바라보는 엔드포인트입니다. `ListProjects`, `GetSession` 등의 RPC를 처리합니다.
- **Repository**: MongoDB와의 통신을 담당합니다. `session_records` 컬렉션 하나를 공유하여 쓰기(Collector)와 읽기(Dashboard)를 수행합니다.

## 3. 구체적인 동작 로직 (Concrete Behavior)

### 데이터 수집 (Collector flow)
1. **Receive**: `SendRecords` RPC 요청 수신.
2. **Validate**: 필수 필드(UUID, SessionID 등) 확인.
3. **Persist**: `SessionRecordRepository.SaveBatch` 호출 -> MongoDB `insertMany` 실행.
   - 이때, 데이터는 `daemon_id`와 함께 저장되어 어느 머신에서 왔는지 식별 가능합니다.

### 데이터 조회 (Dashboard flow)
1. **List Projects**:
   - MongoDB Aggregation Pipeline을 실행합니다.
   - `session_records`를 `daemon_id` 또는 `project_path`로 그룹화(Group)합니다.
   - 각 그룹의 최신 활동 시간(`max(timestamp)`), 총 세션 수(`count(distinct session_id)`), 총 토큰 사용량(`sum(usage.total_tokens)`)을 계산하여 반환합니다.
   
2. **Session Detail**:
   - 특정 `session_id`를 가진 모든 레코드를 시간순으로 정렬하여 조회합니다.
   - 채팅 UI를 구성하기 위해 `User` 메시지와 `Assistant` 메시지, 그리고 `Tool Use` 결과를 구조화하여 반환합니다.
