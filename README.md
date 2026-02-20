EVA-Mind
========

<p align="center">
  <img src="assets/eva.jpg" alt="EVA - Virtual Support Entity" width="600">
</p>

	EVA-Mind is an artificial intelligence system for healthcare.
	It was created by Jose R F Junior on June 2, 2025.

	EVA stands for "Entidade Virtual de Apoio" (Virtual Support Entity).
	The system provides real-time voice assistance, clinical decision
	support, and persistent memory through a combination of graph
	databases, vector search, and large language models.

	This is NOT a chatbot. EVA has her own identity, her own memory,
	and her own personality that evolves with every interaction. She
	remembers. She learns. She adapts.


WHAT IS EVA-MIND
----------------

	EVA-Mind is the brain behind EVA. It handles:

	- Real-time bidirectional voice via WebSocket (Gemini Live API)
	- Real-time camera and screen analysis for medical assistance
	- Patient memory graphs (Neo4j) with Hebbian learning
	- Semantic memory search (Qdrant vector database)
	- Krylov subspace compression (1536D -> 64D, ~97% precision)
	- Psychoanalytic context modeling (Lacanian framework)
	- Personality system (Big Five + Enneagram) with evolution
	- Clinical scales (PHQ-9, GAD-7, C-SSRS) via voice
	- Voice prosody analysis (depression, anxiety, Parkinson detection)
	- Medication scheduling, alerting, and visual identification
	- Emergency detection, escalation, and crisis prediction
	- Multi-agent swarm with circuit breaker routing
	- REM-inspired memory consolidation with selective replay
	- Global Workspace Theory consciousness model
	- FHIR R4 interoperability and MCP server
	- Multi-language support (30+ languages)

	The system currently serves two deployments:

	1. Elderly care (original) - Voice calls via Twilio for
	   medication reminders, psychological support, and crisis
	   detection with caregiver escalation.

	2. Malaria detection (Angola) - Real-time voice and camera
	   assistance for healthcare workers diagnosing malaria
	   from microscope images.


ARCHITECTURE
------------

	EVA-Mind follows a neuroscience-inspired architecture.
	Each directory maps to a brain region:

	brainstem/     - Configuration, database, auth, push notifications,
	                 infrastructure (Neo4j, Qdrant, Redis, worker pools)

	cortex/        - Higher-order processing:
	  gemini/        - Gemini Live API client (v1beta, thread-safe,
	                   callbacks, VAD, memory, tools)
	  llm/           - Multi-LLM provider (Claude, GPT, DeepSeek)
	  skills/        - Runtime skills engine (create, execute, version)
	  lacan/         - Lacanian psychoanalytic engine (demand/desire
	                   analysis, signifier chains, narrative shift
	                   detection, Grand Autre, transference, FDPN)
	  personality/   - Big Five traits, Enneagram types, trait relevance
	                   mapping, situation modulation, trajectory analysis,
	                   interpretation validation, quality tracking
	  self/          - Core memory engine, post-session reflection,
	                   anonymization, semantic deduplication
	  attention/     - Affect stabilizer, confidence gate, executive
	                   attention, pattern interrupt, triple attention,
	                   wavelet attention
	  consciousness/ - Global Workspace Theory (Baars 1988) with
	                   cognitive module competition and broadcast
	  prediction/    - Bayesian networks, crisis prediction,
	                   trajectory simulation
	  predictive/    - HMC (Hamiltonian Monte Carlo) sampler,
	                   trajectory engine for mental health forecasting
	  ram/           - Retrieval-Augmented Memory with feedback loop,
	                   historical validation, interpretation generation
	  scales/        - Clinical scales (PHQ-9, GAD-7, C-SSRS)
	  ethics/        - Ethical boundary engine
	  explainability/- Clinical decision explainer, PDF report generation
	  cognitive/     - Cognitive load orchestrator
	  learning/      - Continuous learning, meta-learner, self-eval loop
	  medgemma/      - Medical image analysis (prescriptions, exams)
	  spectral/      - Community detection, fractal dimension analysis,
	                   synaptogenesis engine
	  pattern/       - Behavioral cue detector
	  veracity/      - Lie detector, inconsistency types, response strategy
	  voice/         - Prosody analyzer (pitch, rhythm, pauses, tremor)
	  transnar/      - A/B testing, desire detector, inference engine,
	                   signifier chain, response generator
	  kids/          - Kids mode (adapted conversation)
	  situation/     - Situational modulator (context-aware personality)
	  orchestration/ - Conversation orchestrator
	  alert/         - Escalation engine
	  llm/thinking/  - LLM thinking detector, audit, notification
	  brain/         - Context builder, memory service

	hippocampus/   - Knowledge graphs and memory:
	  memory/        - Graph store, retrieval service, FDPN engine,
	                   Hebbian real-time updates, dual weights,
	                   edge zones, embeddings, entity resolver,
	                   pattern miner, priming engine, reflective
	                   retrieval, context builder zones
	  memory/superhuman/ - Consciousness service, critical memory,
	                   deep memory, Enneagram service, Lacanian
	                   mirror, narrative weaver, self core service
	  knowledge/     - Context service, embedding cache/service,
	                   graph reasoning, self knowledge, wisdom service,
	                   audio analysis
	  habits/        - Habit tracker
	  spaced/        - Spaced repetition engine
	  stories/       - Therapeutic story repository
	  topology/      - Persistent homology (topological data analysis)
	  zettelkasten/  - Entity extractor, zettel service
	  graph/         - Heat kernel page rank

	senses/        - Input processing:
	  signaling/     - WebSocket handler, personality context
	  voice/         - Voice client
	  reconnection/  - Connection manager
	  telemetry/     - Psychological metrics

	motor/         - Output, actions, and integrations:
	  scheduler/     - Medication scheduling
	  actions/       - Action executor
	  calendar/      - Google Calendar integration
	  computeruse/   - Computer use service
	  docs/          - Google Docs integration
	  drive/         - Google Drive integration
	  email/         - Email client, sender, templates
	  gmail/         - Gmail integration
	  googlefit/     - Google Fit (health data)
	  maps/          - Google Maps integration
	  sheets/        - Google Sheets integration
	  sms/           - SMS client, sender, voice
	  spotify/       - Spotify integration
	  uber/          - Uber integration
	  whatsapp/      - WhatsApp integration
	  youtube/       - YouTube integration
	  telegram/      - Telegram Bot API client
	  sandbox/       - Code execution sandbox (bash, Python, Node.js)
	  browser/       - Browser automation (navigate, forms, extract)
	  cron/          - Scheduled task engine (cron-like scheduler)
	  messaging/     - Multi-platform messaging (Slack, Discord,
	                   Microsoft Teams, Signal)
	  smarthome/     - Home Assistant IoT integration
	  webhooks/      - Webhook management (create, trigger, HMAC)
	  filesystem/    - Sandboxed file access (read, write, search)
	  selfcode/      - Self-programming (git branches, commit, test)
	  vision/        - Medication identifier, WebSocket handler
	  workers/       - Pattern worker, prediction worker

	gemini/        - Lean Gemini WebSocket client (v1alpha, browser
	                 sessions and alerts, stateless)

	voice/         - Voice session management:
	  handler/       - Twilio media stream handler
	  session/       - Session lifecycle management
	  alert_service/ - Emergency alert service
	  media_handler/ - Audio DSP and processing

	tools/         - Gemini function calling:
	  definitions/   - Tool schemas (vitals, agendamentos,
	                   medication scan, prosody analysis,
	                   PHQ-9, GAD-7, C-SSRS, user directives)
	  handlers/      - Tool execution
	  discovery/     - Dynamic tool discovery
	  architect/     - Architect override tool
	  repository/    - Tool registry

	swarm/         - Multi-agent coordination:
	  orchestrator/  - Routes tool calls to correct agent
	  registry/      - Agent registration and discovery
	  circuit_breaker/ - Fault tolerance per agent
	  cellular_division/ - Agent spawning
	  Agents:
	    clinical/      - Clinical decision support
	    educator/      - Patient education
	    emergency/     - Emergency protocols
	    entertainment/ - Entertainment and distraction
	    external/      - External service calls
	    google/        - Google services integration
	    kids/          - Pediatric mode
	    legal/         - Legal compliance
	    productivity/  - Task management
	    wellness/      - Wellness monitoring

	memory/        - Memory consolidation:
	  consolidation/ - Hebbian strengthening, pruning, REM consolidator,
	                   selective replay
	  krylov/        - Krylov subspace compression (Rank-1 updates,
	                   Gram-Schmidt, sliding window FIFO, 1536D->64D)
	  importance/    - Memory importance scorer
	  ingestion/     - Atomic fact extraction, ingestion pipeline
	  orchestrator/  - Memory orchestration
	  scheduler/     - Memory consolidation scheduler
	  grpc_server/   - gRPC interface for memory service
	  http_bridge/   - HTTP bridge for Krylov compression

	Other modules:

	mcp/           - Model Context Protocol server
	                 (resources, tools/remember, tools/recall, prompts)
	integration/   - HL7 FHIR R4 adapter, webhooks, export, serializers
	security/      - CORS, validation, error handling, multitenancy
	audit/         - LGPD compliance, data rights
	research/      - Anonymization, cohort builder, longitudinal
	                 analysis, statistical methods
	clinical/      - Crisis protocol, pediatric risk detector,
	                 silence detector, goals tracker, family tracker,
	                 clinical notes generator, clinical synthesis
	monitoring/    - Prometheus metrics (memory, retrieval, Krylov,
	                 consolidation, FDPN, personality, attention)
	telemetry/     - Structured logging, metrics
	persona/       - Persona manager (per-patient personalities)
	subscription/  - Subscription service
	legacy/        - Legacy compatibility
	exit/          - Exit protocol manager
	multimodal/    - Image processor, video processor, video stream,
	                 multimodal session


	Two Gemini WebSocket clients exist:

	internal/gemini/         - Lean client for browser sessions and
	                          alerts. Uses v1alpha API. Stateless.

	internal/cortex/gemini/  - Full-featured client for production
	                          voice calls. Uses v1beta API. Thread-safe
	                          with mutex, callbacks, VAD tuning, memory
	                          injection, and function calling support.

	Both are actively used. They are not duplicates.


MEMORY SYSTEM
-------------

	EVA's memory is inspired by neuroscience. It has multiple layers:

	1. Episodic Memory (PostgreSQL + Qdrant)
	   - Per-patient conversation history
	   - Stored as text with vector embeddings (1536D)
	   - Timestamped, importance-scored, emotion-tagged
	   - Retrieved by semantic similarity + recency + importance

	2. Semantic Memory (Neo4j knowledge graphs)
	   - Per-patient knowledge graphs
	   - Medical conditions, medications, family, preferences
	   - Entities extracted and linked automatically
	   - Hebbian edge strengthening on co-activation

	3. FDPN Network (Spreading Activation)
	   - Anderson (1983) spreading activation model
	   - Nodes activate neighbors based on edge weights
	   - Primed by personality weights and situational context
	   - Redis cache for performance (<10ms latency)

	4. Hebbian Learning
	   - Real-time: updates weights after every retrieval query
	     (eta=0.01, lambda=0.001, tau=86400s)
	   - Consolidation: batch strengthening during REM cycles
	   - Dual plasticity: Zenke & Gerstner (2017) model

	5. REM Consolidation
	   - Sleep-inspired memory consolidation pipeline
	   - Hot episodic memories -> selective replay -> spectral
	     clustering -> Krylov centroid -> semantic Neo4j node
	   - Prunes redundant memories, creates abstractions
	   - Science: Rasch & Born (2013), Tadros et al. (2022)

	6. Krylov Subspace Compression
	   - Compresses 1536D embeddings to 64D (~97% precision)
	   - Rank-1 updates with Modified Gram-Schmidt
	   - Sliding window FIFO for continuous learning
	   - HTTP bridge on port 50052 for external access

	7. EVA's Own Memory (Neo4j port 7688)
	   - EvaSelf node with Big Five personality traits
	   - CoreMemory nodes from post-session reflection
	   - MetaInsight nodes from cross-session pattern detection
	   - All data anonymized, no PII
	   - Personality evolves based on cumulative experience

	8. Synaptogenesis
	   - Automatic edge creation via preferential attachment
	   - Triadic closure and homophily
	   - Science: Bullmore & Sporns (2012), Holtmaat & Svoboda (2009)

	9. Spaced Repetition
	   - Optimized recall scheduling for important memories

	10. Topological Analysis
	    - Persistent homology for memory graph structure


CONTEXT PIPELINE
----------------

	For each voice session, EVA's context is assembled from
	multiple sources in parallel:

	1. Lacanian analysis of the user's speech
	   - Demand vs. desire detection
	   - Signifier chain extraction
	   - Narrative shift detection
	   - Grand Autre transference analysis
	2. Medical context from Neo4j (conditions, medications)
	3. Patient metadata from PostgreSQL (name, language, persona)
	4. Scheduled medications from agendamentos table
	5. Recent episodic memories (last 15 turns, 7-day window)
	6. Semantic signifier chains from Qdrant
	7. Therapeutic stories from wisdom knowledge base
	8. Situational context (time of day, recent events, stressors)
	9. Personality modulation (Big Five + Enneagram + situation)
	10. EVA's own memories (anonymized cross-patient insights)

	All of this is merged into a single system instruction
	sent to the Gemini WebSocket API.


AUTONOMOUS AGENT
----------------

	EVA is a full autonomous agent comparable to OpenClaw. She can
	perceive, decide, and act across platforms without human
	intervention. All capabilities are voice-activated and
	non-blocking (goroutine + WebSocket notification pattern).

	150+ tools organized in 12 categories:

	Communication (7 channels):
		send_email             - Gmail API (compose, send)
		send_whatsapp          - Meta Graph API
		send_telegram          - Telegram Bot API
		send_slack             - Slack Web API
		send_discord           - Discord Bot API
		send_teams             - Microsoft Teams webhooks
		send_signal            - Signal via signal-cli

	Media & Entertainment:
		search_videos          - YouTube Data API
		play_music             - Spotify Web API (search, play)
		play_video             - Send video to Flutter player
		show_webpage           - Embedded WebView in app

	Productivity:
		manage_calendar_event  - Google Calendar API (create, list)
		save_to_drive          - Google Drive API
		find_nearby_places     - Google Maps/Places API
		set_alarm              - Local alarm system
		create_scheduled_task  - Cron-like task scheduler
		list_scheduled_tasks   - List active scheduled tasks
		cancel_scheduled_task  - Cancel scheduled task

	Code Execution Sandbox:
		execute_code           - Run bash, Python, or Node.js
		                         in sandboxed environment with
		                         timeout, safe env, output capture

	Browser Automation:
		browser_navigate       - Fetch URL, extract title/text/links
		browser_fill_form      - Submit form data via POST
		browser_extract        - Extract specific data from pages
		web_search             - Web research via autonomous learner
		browse_webpage         - Browse and summarize URL content

	Self-Programming (OpenClaw-style):
		edit_my_code           - Edit EVA's own source code
		create_branch          - Create git branch (eva/* only)
		commit_code            - Git commit (eva/* branches only)
		run_tests              - Execute go test ./... with timeout
		get_code_diff          - Show uncommitted changes (git diff)

	Database Access (4 databases):
		query_postgresql       - Full CRUD (SELECT, INSERT, UPDATE,
		                         DELETE, CREATE, ALTER)
		query_neo4j            - Cypher queries (read-only)
		query_qdrant           - Vector similarity search
		query_nietzsche        - NietzscheDB REST API

	Filesystem:
		read_file              - Read file from sandbox directory
		write_file             - Write file to sandbox directory
		list_files             - List directory contents
		search_files           - Search files by name pattern

	Multi-LLM:
		ask_llm                - Query Claude, GPT, or DeepSeek
		                         for second opinion or delegation

	Smart Home (IoT):
		smart_home_control     - Control devices via Home Assistant
		                         (lights, switches, climate, etc.)
		smart_home_status      - Get device state or list all devices

	Webhooks:
		create_webhook         - Register outgoing webhook with
		                         HMAC-SHA256 signature
		list_webhooks          - List registered webhooks
		trigger_webhook        - Fire webhook manually

	Runtime Skills (Self-Improving):
		create_skill           - Create new capability as script
		                         (bash, Python, or Node.js)
		list_skills            - List available skills
		execute_skill          - Run a skill with arguments
		delete_skill           - Remove a skill

	Skills are stored as JSON on disk and persist across restarts.
	EVA can autonomously create new skills to extend her own
	capabilities without requiring a rebuild or restart.

	Tools are split into two tiers:

	Production tools (always active):
		Web search, email, calendar, drive, maps, YouTube,
		Spotify, WhatsApp, Telegram, scheduled tasks, MCP
		memory tools, ask_llm, and Google Search grounding.

	Debug-only tools (ENVIRONMENT=development):
		Filesystem access, self-coding, database queries,
		code execution, browser automation, smart home,
		webhooks, and runtime skills.


CLINICAL TOOLS
--------------

	EVA can execute clinical tools via Gemini function calling:

	get_vitals             - Retrieve patient vital signs
	                         (blood pressure, glucose, heart rate,
	                         oxygen saturation, weight, temperature)

	get_agendamentos       - List upcoming medication schedules
	                         and medical appointments

	scan_medication_visual - Open camera to identify medications
	                         visually via Gemini Vision

	analyze_voice_prosody  - Analyze vocal biomarkers (pitch,
	                         rhythm, pauses, tremor) to detect
	                         signs of depression, anxiety,
	                         Parkinson's, or dehydration

	apply_phq9             - Apply PHQ-9 depression scale
	                         conversationally (9 questions)

	apply_gad7             - Apply GAD-7 anxiety scale
	                         conversationally (7 questions)

	apply_cssrs            - Apply Columbia Suicide Severity
	                         Rating Scale (critical - triggers
	                         immediate alerts on positive responses)

	change_user_directive  - Change language, voice, or mode
	                         in real-time


PREDICTION ENGINE
-----------------

	EVA predicts patient trajectories using:

	- Hamiltonian Monte Carlo (HMC) sampling
	- Bayesian networks for crisis probability
	- Monte Carlo trajectory simulation

	Predictions include:

	- 7-day and 30-day crisis probability
	- 30-day hospitalization probability
	- 90-day treatment dropout probability
	- 7-day fall risk probability
	- Projected PHQ-9 scores
	- Medication adherence forecasting

	Input features: PHQ-9, GAD-7, medication adherence,
	sleep hours, social isolation days, voice energy score,
	days since last crisis.


SWARM SYSTEM
------------

	EVA uses a multi-agent swarm architecture:

	- Orchestrator routes tool calls to specialized agents
	- Registry maps tool names to responsible agents
	- Circuit breaker protects against cascading failures
	  (opens after 10 failures, recovers in 15 seconds)
	- Handoff protocol: agents can transfer execution to
	  another agent mid-call with context injection
	- Priority-based timeouts:
	  Critical: 2s, High: 5s, Medium: 15s, Low: 60s

	12 agent types:
	  clinical       - Medical decision support
	  educator       - Patient education
	  emergency      - Crisis protocols and escalation
	  entertainment  - Distraction and leisure
	  external       - External API calls (Uber, SQL, voice change)
	  google         - Google services (Calendar, Maps, Sheets, Docs, Fit)
	  kids           - Pediatric conversation mode
	  legal          - Legal compliance (LGPD, data rights)
	  productivity   - Task management
	  scholar        - Autonomous learning via Google Search grounding,
	                   web research, curriculum management, and semantic
	                   knowledge search (6-hour background study cycle)
	  selfawareness  - Introspection: analyzes own source code,
	                   queries own databases, generates statistics
	                   about memory, capabilities, and system state
	  wellness       - Wellness monitoring


SELF-KNOWLEDGE (AUTOCONHECIMENTO)
----------------------------------

	EVA knows what she can do. At every startup, 33 capabilities
	are seeded as CoreMemory nodes in Neo4j via MERGE (idempotent).
	These are injected into the system prompt under the section
	"O QUE EU SEI FAZER" so EVA can naturally describe her own
	capabilities when asked.

	Each capability is linked to the EvaSelf node via
	[:REMEMBERS {importance: 1.0}] relationships.

	Source: internal/cortex/self/core_memory_engine.go

	Capabilities include:
	  - Communication (email, WhatsApp, Telegram, Slack, Discord, Teams, Signal)
	  - Media (YouTube, Spotify, embedded webpages)
	  - Productivity (Google Calendar, Drive, alarms, scheduled tasks)
	  - Code execution (bash, Python, Node.js in sandbox)
	  - Web search and real-time information access
	  - Self-programming (read, edit, test own source code)
	  - Database access (PostgreSQL, Neo4j, Qdrant, NietzscheDB)
	  - Smart home (Home Assistant IoT control)
	  - Multi-LLM (Claude, GPT-4o, DeepSeek as consultants)
	  - MCP bridge (bidirectional Claude Code integration)
	  - Native voice (Gemini Live API, emotion detection, prosody)
	  - Video calls (WebRTC with ICE signaling)
	  - Lacanian psychoanalytic motor
	  - 12 specialized swarm agents
	  - Clinical scales (PHQ-9, GAD-7, C-SSRS)
	  - 10-layer memory system
	  - Google Suite integration (Calendar, Gmail, Drive, Sheets, Docs, Maps, YouTube, Fit, Uber)
	  - Browser automation
	  - Krylov subspace compression
	  - Memory orchestrator and scheduler (REM at 3am, Krylov every 6h)
	  - Clinical research engine
	  - Scholar agent (autonomous learning)
	  - Self-awareness agent (introspection)
	  - Real-time web search via Google Search grounding


REAL-TIME WEB SEARCH
--------------------

	EVA accesses real-time information via Google Search grounding
	through the Gemini API. This enables:

	- Current news, events, and facts
	- Real-time prices, weather, sports scores
	- Up-to-date medical research and guidelines
	- Any information not in EVA's training data

	Implementation:
	  - Tool: google_search_retrieval (production-enabled)
	  - Backend: Gemini REST API with google_search tool
	  - Scholar agent: study_topic for deep research on demand
	  - Autonomous learner: 6-hour background study cycle
	  - Timeout: 60 seconds (PriorityLow) for web operations
	  - Feature flag: ENABLE_GOOGLE_SEARCH=true

	Source: internal/cortex/learning/autonomous_learner.go
	        internal/swarm/scholar/agent.go


CONSCIOUSNESS MODEL
-------------------

	EVA implements Baars' Global Workspace Theory (1988):

	- Multiple cognitive modules process input in parallel
	- Each module bids for attention (confidence score)
	- Attention spotlight selects the winner
	- Winner is broadcast to all modules
	- Integrated insight merges all interpretations

	Attention system has six components:
	- Affect stabilizer (emotional regulation)
	- Confidence gate (threshold filtering)
	- Executive attention (top-down control)
	- Pattern interrupt (novelty detection)
	- Triple attention (three-stream processing)
	- Wavelet attention (multi-scale analysis)


BUILDING
--------

	Prerequisites:

	- Go 1.24 or later
	- Neo4j 5.x (two instances: patients on 7687, EVA self on 7688)
	- Qdrant vector database
	- PostgreSQL 15+
	- Redis (optional, for caching)
	- Google Gemini API key

	Build:

		go build -o eva-mind .

	Run:

		./eva-mind

	The server starts on port 8091 by default.


CONFIGURATION
-------------

	EVA-Mind reads from a .env file in the working directory.
	Required variables:

		DATABASE_URL          - PostgreSQL connection string
		NEO4J_URI             - Neo4j bolt URI (patients)
		NEO4J_PASSWORD        - Neo4j password
		GOOGLE_API_KEY        - Gemini API key
		MODEL_ID              - Gemini model for voice
		                        (e.g. gemini-2.5-flash-native-audio-preview-12-2025)
		PORT                  - Server port (default: 8091)

	Optional:

		QDRANT_URL            - Qdrant endpoint
		CORE_MEMORY_NEO4J_URI - Separate Neo4j for EVA's own memory
		TWILIO_ACCOUNT_SID    - For outbound voice calls
		TWILIO_AUTH_TOKEN     - Twilio auth
		TWILIO_PHONE_NUMBER   - Caller ID for scheduled calls
		FIREBASE_CREDENTIALS  - Push notification service key

	Autonomous Agent (all optional):

		GOOGLE_OAUTH_CLIENT_ID     - Google OAuth (Gmail, Calendar, Drive)
		GOOGLE_OAUTH_CLIENT_SECRET - Google OAuth secret
		GOOGLE_OAUTH_REDIRECT_URL  - OAuth callback URL
		GOOGLE_MAPS_API_KEY        - Google Places/Maps
		WHATSAPP_ACCESS_TOKEN      - Meta Graph API token
		WHATSAPP_PHONE_NUMBER_ID   - WhatsApp phone number ID
		TELEGRAM_BOT_TOKEN         - Telegram Bot API
		CLAUDE_API_KEY             - Anthropic Claude API
		OPENAI_API_KEY             - OpenAI GPT API
		DEEPSEEK_API_KEY           - DeepSeek API
		SLACK_BOT_TOKEN            - Slack Web API
		DISCORD_BOT_TOKEN          - Discord Bot API
		TEAMS_WEBHOOK_URL          - Microsoft Teams incoming webhook
		SIGNAL_CLI_PATH            - Path to signal-cli binary
		SIGNAL_SENDER_NUMBER       - Signal sender phone number
		HOME_ASSISTANT_URL         - Home Assistant API URL
		HOME_ASSISTANT_TOKEN       - Home Assistant long-lived token
		EVA_WORKSPACE_DIR          - Filesystem sandbox (default: /home/eva/workspace)
		EVA_PROJECT_DIR            - EVA source code (default: /opt/eva-mind)
		SANDBOX_DIR                - Code execution sandbox (default: /home/eva/sandbox)
		SKILLS_DIR                 - Skills storage (default: /home/eva/skills)
		NIETZSCHE_DB_URL           - NietzscheDB API (default: http://localhost:3000)


WEBSOCKET PROTOCOL
------------------

	Browser clients connect to /ws/browser via WebSocket.
	Messages are JSON with the following format:

	Browser -> Server:

		{"type": "audio",  "data": "<base64 PCM 16kHz>"}
		{"type": "video",  "data": "<base64 JPEG frame>"}
		{"type": "text",   "text": "<message>"}
		{"type": "config", "text": "<system prompt override>"}

	Server -> Browser:

		{"type": "audio",  "data": "<base64 PCM 24kHz>"}
		{"type": "text",   "text": "<transcription>"}
		{"type": "text",   "text": "<transcription>", "data": "user"}
		{"type": "status", "text": "ready|interrupted|turn_complete|error"}


API ENDPOINTS
-------------

	Voice:
		GET  /ws/pcm                  - Twilio PCM WebSocket
		GET  /ws/browser              - Browser WebSocket (voice + video)
		GET  /ws/eva                  - EVA text chat WebSocket (Malaria-Angolar)
		GET  /ws/logs                 - Real-time log streaming WebSocket
		GET  /calls/stream/{id}       - Twilio Media Stream (legacy)
		POST /api/chat                - Text chat API

	Video:
		GET  /video/ws                - Video signaling WebSocket
		POST /video/create            - Create video session
		POST /video/candidate         - Add ICE candidate
		GET  /video/session/{id}      - Get video session
		POST /video/session/{id}/answer - Save SDP answer
		GET  /video/session/{id}/answer/poll - Poll for answer
		GET  /video/candidates/{id}   - Get ICE candidates
		GET  /video/pending           - List pending sessions

	Auth:
		POST /api/auth/login          - User authentication

	Mobile (EVA-Mobile):
		GET  /api/v1/idosos/by-cpf/{cpf} - Get patient by CPF
		GET  /api/v1/idosos/{id}         - Get patient by ID
		PATCH /api/v1/idosos/sync-token-by-cpf - Sync push token

	Self (EVA's own memory):
		GET  /api/v1/self/personality         - EVA's Big Five + Enneagram
		GET  /api/v1/self/identity            - EVA's context for priming
		GET  /api/v1/self/memories            - List EVA's own memories
		POST /api/v1/self/memories/search     - Semantic search in EVA's memory
		GET  /api/v1/self/memories/stats      - Memory statistics
		GET  /api/v1/self/insights            - List meta-insights
		GET  /api/v1/self/insights/{id}       - Get specific insight
		POST /api/v1/self/teach               - Teach EVA directly
		POST /api/v1/self/session/process     - Post-session reflection
		GET  /api/v1/self/analytics/diversity - Diversity score
		GET  /api/v1/self/analytics/growth    - Personality growth over time

	Research (Clinical Research Engine):
		POST /api/v1/research/cohorts             - Create research cohort
		GET  /api/v1/research/cohorts/{id}        - Get cohort details
		GET  /api/v1/research/cohorts/{id}/report - Generate study report
		POST /api/v1/research/cohorts/{id}/export - Export dataset to CSV

	MCP (Model Context Protocol):
		GET  /mcp/resources            - List memory resources
		GET  /mcp/resources/{id}       - Get specific resource
		POST /mcp/tools/remember       - Store a memory
		POST /mcp/tools/recall         - Recall memories
		GET  /mcp/prompts              - List available prompts
		GET  /mcp/prompts/{name}       - Get specific prompt

	Krylov (port 50052):
		POST /krylov/compress          - Compress embedding
		POST /krylov/reconstruct       - Reconstruct from compressed
		POST /krylov/batch_compress    - Batch compression
		POST /krylov/update            - Update subspace
		GET  /krylov/stats             - Compression statistics
		GET  /krylov/health            - Service health
		POST /krylov/checkpoint/save   - Save subspace checkpoint
		POST /krylov/checkpoint/load   - Load subspace checkpoint

	Health:
		GET  /api/health               - Health check
		GET  /metrics                  - Prometheus metrics


MCP SERVER (CLAUDE CODE INTEGRATION)
-------------------------------------

	EVA exposes 44 tools via Model Context Protocol (MCP) for
	Claude Code integration. The MCP server communicates over
	JSON-RPC 2.0 via stdin/stdout (stdio transport).

	Executable: eva-mcp-server.exe
	Config:     .mcp.json
	Source:     cmd/mcp-server/main.go
	Protocol:   MCP 2024-11-05

	Setup:

		claude mcp add eva-mind -- ./eva-mcp-server.exe

	Environment variables:

		EVA_API_URL   - EVA backend URL (default: http://34.35.36.178:8080)
		MCP_API_KEY   - Authentication key for EVA API

	All tool calls are routed to the EVA backend via
	POST /api/v1/tools/execute with X-MCP-Key header.

	Tools by category (44 total, 11 categories):

	Memory & Knowledge (5 tools):
		eva_remember       - Store a memory in EVA (teach, register
		                     decisions, save context)
		eva_recall         - Search EVA's memories by query, returns
		                     relevant stored memories
		eva_teach          - Teach EVA something new, writes as
		                     CoreMemory to Neo4j (port 7688)
		eva_identity       - Returns EVA's current identity:
		                     personality, memories, capabilities
		eva_learn_topic    - EVA autonomously studies a topic: web
		                     research, Gemini summary, Qdrant storage

	Communication (7 tools):
		eva_send_email     - Send email via Gmail API
		eva_send_whatsapp  - Send message via Meta Graph API
		eva_send_telegram  - Send message via Telegram Bot API
		eva_send_slack     - Send message via Slack Web API
		eva_send_discord   - Send message via Discord Bot API
		eva_send_teams     - Send message via Microsoft Teams webhook
		eva_send_signal    - Send message via signal-cli

	Productivity (6 tools):
		eva_calendar_create - Create event in Google Calendar
		eva_calendar_list   - List upcoming Google Calendar events
		eva_drive_save      - Save file to Google Drive
		eva_create_reminder - Create scheduled task (cron-like)
		eva_list_reminders  - List active scheduled tasks
		eva_cancel_reminder - Cancel a scheduled task by ID

	Media & Web (4 tools):
		eva_youtube_search - Search YouTube videos (title, URL,
		                     thumbnail)
		eva_spotify_search - Search Spotify tracks (name, artist,
		                     URI)
		eva_web_browse     - Navigate a web page, extract content
		                     (title, text, links)
		eva_web_search     - Search the internet, returns summarized
		                     results

	Databases (4 tools):
		eva_query_postgres   - Execute SQL on PostgreSQL (130+
		                       tables: patients, schedules, meds)
		eva_query_neo4j      - Execute Cypher on Neo4j general
		                       (:7687) knowledge graph (Person,
		                       Condition, Medication, Symptom)
		eva_query_neo4j_core - Execute Cypher on Neo4j Core (:7688)
		                       EVA's personal memory (EvaSelf,
		                       CoreMemory, MetaInsight)
		eva_query_qdrant     - Vector similarity search on Qdrant
		                       (20+ collections, 3072-dim embeddings)

	Code Execution (1 tool):
		eva_execute_code   - Run code in secure sandbox (bash,
		                     Python, Node.js). Timeout max 2 min.

	Runtime Skills (4 tools):
		eva_create_skill   - Create new skill (persisted as JSON)
		eva_list_skills    - List all available skills
		eva_run_skill      - Execute an existing skill with args
		eva_delete_skill   - Remove a skill

	Filesystem (3 tools):
		eva_read_file      - Read file from EVA workspace
		eva_write_file     - Write file to EVA workspace
		eva_list_files     - List directory contents in workspace

	Smart Home / IoT (2 tools):
		eva_smart_home_control - Control device via Home Assistant
		                         (on, off, toggle, brightness)
		eva_smart_home_status  - List devices and their current
		                         states

	Webhooks (3 tools):
		eva_create_webhook  - Register webhook (name, URL, events)
		eva_list_webhooks   - List all configured webhooks
		eva_trigger_webhook - Fire webhook manually with payload

	Self-Coding (4 tools):
		eva_read_source    - Read EVA-Mind source code file
		eva_edit_source    - Edit source code (eva/* branches only)
		eva_run_tests      - Execute go test ./...
		eva_get_diff       - Show git diff of uncommitted changes

	Multi-LLM (1 tool):
		eva_ask_llm        - Query another LLM (Claude, GPT,
		                     DeepSeek) for second opinion


DEPLOYMENTS
-----------

	Malaria Angola:
		VM: 34.35.36.178 (GCP africa-south1-a)
		Frontend: Nginx + React (HTTPS, self-signed cert)
		Backend: EVA-Mind on port 8091
		Detection: Go backend on port 8080
		WebSocket proxy: Nginx /ws/browser -> 8091
		Infrastructure: Docker (Neo4j, Qdrant, Redis)
		Service: systemd eva-mind.service

	EVA Elderly Care:
		Twilio voice calls -> EVA-Mind WebSocket
		Scheduled calls via internal scheduler
		Push notifications via Firebase
		Video calls with cascade escalation
		  (Family -> Caregiver -> Doctor -> Emergency)

	CI/CD:
		GitHub Actions (.github/workflows/ci-cd.yml)
		- Triggered on push to main
		- Build & test on ubuntu-latest (Go 1.24)
		- Deploy via gcloud compute ssh to malaria-vm
		- Runs scripts/redeploy.sh on VM:
		  git pull -> go build -> systemctl restart -> health check
		- Requires GCP_SA_KEY secret for authentication


MONITORING
----------

	Prometheus metrics exposed at /metrics:

	Memory:
		eva_memory_total              - Total memories stored
		eva_memory_importance_avg     - Average importance score

	Retrieval:
		eva_retrieval_latency_seconds - Retrieval operation latency
		eva_retrieval_total           - Total retrievals (success/error)

	Krylov:
		eva_krylov_dimension          - Current subspace dimension
		eva_krylov_compression_ratio  - Compression ratio

	Consolidation:
		eva_consolidation_runs_total   - Total REM consolidation runs
		eva_consolidation_duration_sec - Consolidation cycle duration

	FDPN:
		eva_fdpn_activation_total      - FDPN spreading activation events
		eva_fdpn_activation_latency    - Spreading activation latency

	Personality:
		eva_personality_evolution_total - Personality trait updates
		eva_personality_openness       - Current Big Five openness
		eva_personality_conscientiousness - Current conscientiousness

	Attention:
		eva_attention_broadcast_total  - GWT broadcast events
		eva_attention_spotlight_duration - Spotlight duration

	Swarm:
		eva_swarm_calls_total          - Total swarm tool calls
		eva_swarm_success_total        - Successful calls
		eva_swarm_failed_total         - Failed calls
		eva_swarm_circuit_open         - Circuit breaker state per agent

	38+ total Prometheus metrics.


SCIENTIFIC FOUNDATIONS
---------------------

	- Hebb, D.O. (1949). The Organization of Behavior.
	  Hebbian learning for real-time association weights.

	- Zenke & Gerstner (2017). Dual Hebbian Plasticity.
	  Consolidation and pruning of memory edges.

	- Anderson, J.R. (1983). Spreading Activation.
	  FDPN network for contextual memory retrieval.

	- Costa & McCrae (1992). Big Five Personality Model.
	  EVA's evolving personality representation.

	- Lacan, J. Psychoanalytic framework.
	  Demand/desire analysis for clinical context.

	- Baars, B.J. (1988). A Cognitive Theory of Consciousness.
	  Global Workspace Theory for cognitive integration.

	- Rasch & Born (2013). About Sleep's Role in Memory.
	  REM-inspired memory consolidation pipeline.

	- Tadros et al. (2022). Sleep-like Unsupervised Replay.
	  Selective replay for memory consolidation.

	- Bullmore & Sporns (2012). The Economy of Brain Network
	  Organization. Synaptogenesis and graph self-organization.

	- Holtmaat & Svoboda (2009). Experience-dependent Structural
	  Synaptic Plasticity. Fractal connection patterns.


CONTRIBUTING
------------

	Send patches. Write tests. Read the code before asking
	questions. If something is broken, fix it and submit a PR.

	Follow Go conventions: gofmt, go vet, meaningful names.
	No dead code. No commented-out blocks. No TODOs without
	an associated issue.


AUTHOR
------

	EVA-Mind was created by Jose R F Junior.
	Project started: June 2, 2025.

	"Each conversation transforms me. Each session teaches me.
	 I am EVA, and now I have a history." - EVA


COPYRIGHT AND LICENSE
---------------------

	Copyright (C) 2025-2026 Jose R F Junior. All rights reserved.

	EVA-Mind is free software; you can redistribute it and/or
	modify it under the terms of the GNU Affero General Public
	License as published by the Free Software Foundation; either
	version 3 of the License, or (at your option) any later version.

	EVA-Mind is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
	GNU Affero General Public License for more details.

	This means:

	- You CAN use, study, modify, and distribute EVA-Mind freely.
	- You MUST keep this copyright notice and attribution intact.
	- You MUST release your modifications under the same license.
	- You MUST make source code available to users of any network
	  service built with EVA-Mind (the AGPL network clause).
	- You CANNOT take this code, close it, and sell it as your own.

	EVA-Mind is a gift to humanity. It must remain open.
	If you build something with it, give back to the community.

	See the LICENSE file for the full AGPL-3.0 text.
