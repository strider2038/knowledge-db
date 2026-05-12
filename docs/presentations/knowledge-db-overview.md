---
title: Knowledge DB
subtitle: LLM, агенты и RAG в локальной базе знаний
author: Igor Lazarev
theme: white
css: knowledge-db-course.css
revealjs-url: https://cdn.jsdelivr.net/npm/reveal.js@5
slideNumber: true
transition: slide
backgroundTransition: fade
---

## LLM, агенты и RAG в локальной базе знаний

<div class="hero-grid">
<div>

<strong>Итоговый проект курса:</strong> персональная база знаний, где LLM не заменяет систему, а становится её рабочим слоем.

</div>
<div class="signal-card">

<code>offline-first</code> · <code>git-first</code> · <code>LLM-optional</code>

</div>
</div>

<div class="footer-note">Igor Lazarev · 2026</div>

---

## Задача проекта

<div class="two-col">
<div>
<h3>Было</h3>
<ul>
<li>поток статей, новостей, идей и ссылок растёт быстрее, чем его получается разбирать</li>
<li>многое оседает в избранном Telegram и потом плохо находится</li>
<li>LLM отвечает без устойчивого личного контекста</li>
<li>статьи и сайты могут исчезать или быть недоступны</li>
</ul>
</div>
<div>
<h3>Стало</h3>
<ul>
<li>локальная база в Markdown под Git</li>
<li>ingestion через LLM-инструменты</li>
<li>гибридный поиск и RAG-чат с источниками</li>
<li>веб-интерфейс, Telegram-бот и массовый импорт как входные каналы</li>
</ul>
</div>
</div>

> Главная идея: LLM работает поверх контролируемой базы знаний, а не вместо неё.

---

## Мотивация

<div class="result-grid">
<div><b>Информационный поток</b><span>Ссылки, статьи, новости и идеи нужно не только сохранять, но и превращать в доступное знание.</span></div>
<div><b>Надёжное хранение</b><span>Git + Markdown дают локальную копию на ПК, ноутбуке или VPS, историю изменений и защиту от исчезающих источников.</span></div>
<div><b>Гибкие режимы</b><span>Локальное чтение работает без сетевых LLM; для AI можно подключать Ollama, LM Studio или OpenAI-compatible API.</span></div>
<div><b>Быстрая запись</b><span>Материалы добавляются из Web UI, Telegram-бота, API и массового импорта сохранённых заметок Telegram.</span></div>
</div>

---

## Два сценария использования

<div class="mode-grid">
<div class="mode">
<b>Локальный</b>
<span>База открывается и читается на рабочей машине без интернета.</span>
<small>Ключевой поиск обязателен, LLM и embeddings опциональны.</small>
</div>
<div class="mode">
<b>Локальный AI</b>
<span>RAG и чат работают через локальные модели.</span>
<small>Ollama для embeddings, LM Studio или другой OpenAI-compatible endpoint для chat.</small>
</div>
<div class="mode">
<b>Self-hosted</b>
<span>Сервер на своей VPS даёт быстрый доступ со смартфона и Telegram ingestion.</span>
<small>Удобно для записи, синхронизации, OAuth и фоновых обработчиков.</small>
</div>
</div>

---

## Что показывает проект

<div class="metric-row">
<div class="metric"><span>1</span><b>LLM orchestration</b><small>function calling, tools, prompts</small></div>
<div class="metric"><span>2</span><b>RAG pipeline</b><small>embeddings, hybrid retrieval, sources</small></div>
<div class="metric"><span>3</span><b>Agent workflows</b><small>OpenSpec, skills, MCP endpoint</small></div>
<div class="metric"><span>4</span><b>Production UX</b><small>streaming, sessions, fallback</small></div>
</div>

<div class="takeaway">
Не учебный playground, а локальный продуктовый прототип с реальными ограничениями: offline-first, git-first, explainable retrieval.
</div>

---

## Карта системы

<svg class="diagram-svg" viewBox="0 0 1120 520" role="img" aria-label="Карта системы Knowledge DB">
<defs>
<marker id="arrow-system" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="7" markerHeight="7" orient="auto-start-reverse"><path d="M 0 0 L 10 5 L 0 10 z" class="diagram-arrow-head"/></marker>
</defs>
<g class="diagram-lane">
<rect x="36" y="48" width="210" height="424" rx="14"/>
<text x="141" y="82" text-anchor="middle">Каналы записи</text>
</g>
<g class="diagram-lane">
<rect x="310" y="48" width="210" height="424" rx="14"/>
<text x="415" y="82" text-anchor="middle">AI-обработка</text>
</g>
<g class="diagram-lane">
<rect x="584" y="48" width="210" height="424" rx="14"/>
<text x="689" y="82" text-anchor="middle">Хранилище</text>
</g>
<g class="diagram-lane">
<rect x="858" y="48" width="226" height="424" rx="14"/>
<text x="971" y="82" text-anchor="middle">Использование</text>
</g>
<g class="diagram-node"><rect x="72" y="118" width="138" height="54" rx="10"/><text x="141" y="151" text-anchor="middle">Web UI</text></g>
<g class="diagram-node"><rect x="72" y="194" width="138" height="54" rx="10"/><text x="141" y="227" text-anchor="middle">Telegram bot</text></g>
<g class="diagram-node"><rect x="72" y="270" width="138" height="54" rx="10"/><text x="141" y="303" text-anchor="middle">Telegram import</text></g>
<g class="diagram-node"><rect x="72" y="346" width="138" height="54" rx="10"/><text x="141" y="379" text-anchor="middle">API / MCP</text></g>
<g class="diagram-node accent"><rect x="342" y="198" width="146" height="86" rx="12"/><text x="415" y="232" text-anchor="middle">LLM ingestion</text><text x="415" y="258" text-anchor="middle">tool loop</text></g>
<g class="diagram-node storage"><rect x="620" y="118" width="138" height="78" rx="12"/><text x="689" y="151" text-anchor="middle">Markdown</text><text x="689" y="177" text-anchor="middle">+ Git</text></g>
<g class="diagram-node"><rect x="620" y="252" width="138" height="78" rx="12"/><text x="689" y="285" text-anchor="middle">SQLite index</text><text x="689" y="311" text-anchor="middle">FTS + vectors</text></g>
<g class="diagram-node"><rect x="890" y="118" width="162" height="70" rx="12"/><text x="971" y="147" text-anchor="middle">Hybrid retrieval</text><text x="971" y="171" text-anchor="middle">RRF + cutoff</text></g>
<g class="diagram-node accent-2"><rect x="890" y="246" width="162" height="70" rx="12"/><text x="971" y="275" text-anchor="middle">RAG Chat</text><text x="971" y="299" text-anchor="middle">SSE + sources</text></g>
<g class="diagram-node"><rect x="890" y="374" width="162" height="54" rx="10"/><text x="971" y="407" text-anchor="middle">Agent access</text></g>
<path class="diagram-link" marker-end="url(#arrow-system)" d="M210 145 C260 145 285 220 342 220"/>
<path class="diagram-link" marker-end="url(#arrow-system)" d="M210 221 C260 221 285 236 342 236"/>
<path class="diagram-link" marker-end="url(#arrow-system)" d="M210 297 C260 297 285 262 342 262"/>
<path class="diagram-link" marker-end="url(#arrow-system)" d="M210 373 C268 373 292 278 342 278"/>
<path class="diagram-link" marker-end="url(#arrow-system)" d="M488 222 C540 190 570 160 620 157"/>
<path class="diagram-link" marker-end="url(#arrow-system)" d="M689 196 L689 252"/>
<path class="diagram-link" marker-end="url(#arrow-system)" d="M758 291 C812 290 838 168 890 153"/>
<path class="diagram-link" marker-end="url(#arrow-system)" d="M971 188 L971 246"/>
<path class="diagram-link" marker-end="url(#arrow-system)" d="M758 157 C818 156 838 388 890 401"/>
</svg>

<div class="caption">Source of truth остаётся простым: файлы. AI-слой можно включать, отключать и переиндексировать.</div>

---

## LLM ingestion: модель как оркестратор

<div class="flow-strip">
<div>Пользовательский ввод</div>
<div>LLM выбирает tool</div>
<div>Контент извлекается системой</div>
<div>LLM создаёт структуру</div>
<div>Markdown + Git commit</div>
</div>

<svg class="diagram-svg sequence" viewBox="0 0 1120 460" role="img" aria-label="Последовательность LLM ingestion">
<defs>
<marker id="arrow-seq" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="7" markerHeight="7" orient="auto-start-reverse"><path d="M 0 0 L 10 5 L 0 10 z" class="diagram-arrow-head"/></marker>
</defs>
<g class="diagram-participant"><rect x="56" y="42" width="160" height="52" rx="10"/><text x="136" y="75" text-anchor="middle">User/API/TG</text></g>
<g class="diagram-participant"><rect x="282" y="42" width="160" height="52" rx="10"/><text x="362" y="75" text-anchor="middle">Pipeline</text></g>
<g class="diagram-participant"><rect x="508" y="42" width="160" height="52" rx="10"/><text x="588" y="75" text-anchor="middle">LLM</text></g>
<g class="diagram-participant"><rect x="734" y="42" width="160" height="52" rx="10"/><text x="814" y="75" text-anchor="middle">Tools</text></g>
<g class="diagram-participant"><rect x="960" y="42" width="160" height="52" rx="10"/><text x="1040" y="75" text-anchor="middle">Knowledge DB</text></g>
<path class="diagram-life" d="M136 94 V420"/><path class="diagram-life" d="M362 94 V420"/><path class="diagram-life" d="M588 94 V420"/><path class="diagram-life" d="M814 94 V420"/><path class="diagram-life" d="M1040 94 V420"/>
<path class="diagram-link" marker-end="url(#arrow-seq)" d="M136 132 H362"/><text class="diagram-label" x="249" y="121" text-anchor="middle">text / URL + hints</text>
<path class="diagram-link" marker-end="url(#arrow-seq)" d="M362 190 H588"/><text class="diagram-label" x="475" y="179" text-anchor="middle">input + topics + vocabulary</text>
<path class="diagram-link" marker-end="url(#arrow-seq)" d="M588 248 H814"/><text class="diagram-label" x="701" y="237" text-anchor="middle">fetch_url_content / meta</text>
<path class="diagram-link dashed" marker-end="url(#arrow-seq)" d="M814 306 H362"/><text class="diagram-label" x="588" y="295" text-anchor="middle">markdown / metadata</text>
<path class="diagram-link" marker-end="url(#arrow-seq)" d="M588 350 H362"/><text class="diagram-label" x="475" y="339" text-anchor="middle">create_node(...)</text>
<path class="diagram-link" marker-end="url(#arrow-seq)" d="M362 394 H1040"/><text class="diagram-label" x="701" y="383" text-anchor="middle">node.md + frontmatter + git commit</text>
</svg>

---

## Function calling: что реально делает LLM

<div class="code-grid">
<div>

<pre><code class="language-text">
tools:
  fetch_url_content(url)
  fetch_url_meta(url)
  create_node(...)
</code></pre>

</div>
<div>

<pre><code class="language-yaml">
title: ...
type: article | link | note
annotation: 2-5 предложений
keywords: [...]
theme_path: ...
source_url: ...
</code></pre>

</div>
</div>

<div class="takeaway">
LLM принимает решения, но не владеет данными: полный контент кешируется и записывается сервером, Git фиксирует результат.
</div>

---

## Retrieval: не только embeddings

<div class="two-col">
<div>
<h3>Почему не “vector search и всё”</h3>
<ul>
<li>точные термины важнее “похожести”</li>
<li>русский и английский часто смешаны</li>
<li>длинная статья не должна побеждать числом чанков</li>
<li>чат не должен отвечать на случайном top-K</li>
</ul>
</div>
<div>
<h3>Что сделано</h3>
<ul>
<li>keyword/FTS + vector node search + vector chunk search</li>
<li>Reciprocal Rank Fusion</li>
<li>exact boosts по title/path/aliases/keywords</li>
<li>relevance cutoff для chat mode</li>
<li>диагностика причин совпадения в UI</li>
</ul>
</div>
</div>

---

## Hybrid retrieval pipeline

<svg class="diagram-svg" viewBox="0 0 1120 520" role="img" aria-label="Hybrid retrieval pipeline">
<defs>
<marker id="arrow-retrieval" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="7" markerHeight="7" orient="auto-start-reverse"><path d="M 0 0 L 10 5 L 0 10 z" class="diagram-arrow-head"/></marker>
</defs>
<g class="diagram-node accent"><rect x="56" y="214" width="156" height="64" rx="12"/><text x="134" y="252" text-anchor="middle">Запрос пользователя</text></g>
<g class="diagram-node decision"><path d="M360 178 L476 246 L360 314 L244 246 Z"/><text x="360" y="242" text-anchor="middle">LLM</text><text x="360" y="266" text-anchor="middle">rewrite?</text></g>
<g class="diagram-node"><rect x="548" y="214" width="172" height="64" rx="12"/><text x="634" y="240" text-anchor="middle">Компактный</text><text x="634" y="264" text-anchor="middle">retrieval query</text></g>
<g class="diagram-node"><rect x="820" y="70" width="176" height="58" rx="10"/><text x="908" y="105" text-anchor="middle">Keyword / FTS</text></g>
<g class="diagram-node"><rect x="820" y="154" width="176" height="58" rx="10"/><text x="908" y="189" text-anchor="middle">Exact boosts</text></g>
<g class="diagram-node"><rect x="820" y="238" width="176" height="58" rx="10"/><text x="908" y="273" text-anchor="middle">Vector nodes</text></g>
<g class="diagram-node"><rect x="820" y="322" width="176" height="58" rx="10"/><text x="908" y="357" text-anchor="middle">Vector chunks</text></g>
<g class="diagram-node accent-2"><rect x="548" y="392" width="172" height="64" rx="12"/><text x="634" y="418" text-anchor="middle">RRF fusion</text><text x="634" y="442" text-anchor="middle">+ scoring</text></g>
<g class="diagram-node"><rect x="316" y="392" width="164" height="64" rx="12"/><text x="398" y="418" text-anchor="middle">Filters</text><text x="398" y="442" text-anchor="middle">+ chat cutoff</text></g>
<g class="diagram-node storage"><rect x="56" y="392" width="172" height="64" rx="12"/><text x="142" y="418" text-anchor="middle">Ranked results</text><text x="142" y="442" text-anchor="middle">sources + fragments</text></g>
<path class="diagram-link" marker-end="url(#arrow-retrieval)" d="M212 246 H244"/>
<path class="diagram-link" marker-end="url(#arrow-retrieval)" d="M476 246 H548"/><text class="diagram-label" x="512" y="235" text-anchor="middle">ok</text>
<path class="diagram-link dashed" marker-end="url(#arrow-retrieval)" d="M360 314 C360 356 196 354 155 280"/><text class="diagram-label" x="257" y="345" text-anchor="middle">fallback</text>
<path class="diagram-link" marker-end="url(#arrow-retrieval)" d="M720 246 C770 246 770 99 820 99"/>
<path class="diagram-link" marker-end="url(#arrow-retrieval)" d="M720 246 C770 246 770 183 820 183"/>
<path class="diagram-link" marker-end="url(#arrow-retrieval)" d="M720 246 H820"/>
<path class="diagram-link" marker-end="url(#arrow-retrieval)" d="M720 246 C770 246 770 351 820 351"/>
<path class="diagram-link" marker-end="url(#arrow-retrieval)" d="M908 128 C908 424 772 424 720 424"/>
<path class="diagram-link" marker-end="url(#arrow-retrieval)" d="M908 212 C908 424 772 424 720 424"/>
<path class="diagram-link" marker-end="url(#arrow-retrieval)" d="M908 296 C908 424 772 424 720 424"/>
<path class="diagram-link" marker-end="url(#arrow-retrieval)" d="M908 380 C900 424 772 424 720 424"/>
<path class="diagram-link" marker-end="url(#arrow-retrieval)" d="M548 424 H480"/>
<path class="diagram-link" marker-end="url(#arrow-retrieval)" d="M316 424 H228"/>
</svg>

<div class="caption">LLM rewrite получает vocabulary hints из локального индекса и безопасно падает обратно на исходный запрос.</div>

---

## RAG chat: ответ с основанием

<div class="demo-frame">

<pre><code class="language-text">
User: Что есть в базе про RAG?

Retrieval:
  sources:
    - docs/research/rag.md
    - ai/agentic-coding/...
  fragments:
    - "Progressive SQLite..."
    - "Reciprocal Rank Fusion..."

Assistant:
  отвечает по контексту базы и показывает источники
</code></pre>

</div>

<div class="takeaway">
Ответ не должен быть “магическим”: пользователь видит источники, фрагменты и причины, почему они попали в контекст.
</div>

---

## Три режима чата

<div class="mode-grid">
<div class="mode">
<b>chat_memory</b>
<span>Вопросы о текущем диалоге</span>
<small>без обращения к базе</small>
</div>
<div class="mode">
<b>rag_kb</b>
<span>Явные вопросы по базе знаний</span>
<small>строгий контекст + safe fallback</small>
</div>
<div class="mode">
<b>hybrid</b>
<span>Обычные уточнения</span>
<small>история диалога + RAG-контекст</small>
</div>
</div>

<div class="caption">Маршрутизация режима снижает шум: “резюмируй чат” не запускает поиск, а “что есть в базе” не заставляет LLM фантазировать.</div>

---

## Память диалога

<svg class="diagram-svg" viewBox="0 0 1120 360" role="img" aria-label="Память диалога">
<defs>
<marker id="arrow-memory" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="7" markerHeight="7" orient="auto-start-reverse"><path d="M 0 0 L 10 5 L 0 10 z" class="diagram-arrow-head"/></marker>
</defs>
<g class="diagram-node accent"><rect x="56" y="140" width="150" height="62" rx="12"/><text x="131" y="177" text-anchor="middle">Chat session</text></g>
<g class="diagram-node"><rect x="274" y="130" width="170" height="82" rx="12"/><text x="359" y="162" text-anchor="middle">User/assistant</text><text x="359" y="188" text-anchor="middle">messages</text></g>
<g class="diagram-node decision"><path d="M590 100 L706 171 L590 242 L474 171 Z"/><text x="590" y="168" text-anchor="middle">Prompt</text><text x="590" y="192" text-anchor="middle">budget?</text></g>
<g class="diagram-node"><rect x="780" y="72" width="170" height="62" rx="12"/><text x="865" y="109" text-anchor="middle">Recent history</text></g>
<g class="diagram-node"><rect x="780" y="204" width="170" height="62" rx="12"/><text x="865" y="230" text-anchor="middle">Service</text><text x="865" y="254" text-anchor="middle">summary</text></g>
<g class="diagram-node storage"><rect x="1000" y="140" width="110" height="62" rx="12"/><text x="1055" y="177" text-anchor="middle">LLM</text></g>
<path class="diagram-link" marker-end="url(#arrow-memory)" d="M206 171 H274"/>
<path class="diagram-link" marker-end="url(#arrow-memory)" d="M444 171 H474"/>
<path class="diagram-link" marker-end="url(#arrow-memory)" d="M706 171 C742 171 742 103 780 103"/><text class="diagram-label" x="742" y="130" text-anchor="middle">ok</text>
<path class="diagram-link" marker-end="url(#arrow-memory)" d="M706 171 C742 171 742 235 780 235"/><text class="diagram-label" x="746" y="220" text-anchor="middle">too large</text>
<path class="diagram-link" marker-end="url(#arrow-memory)" d="M950 103 C982 103 974 171 1000 171"/>
<path class="diagram-link" marker-end="url(#arrow-memory)" d="M950 235 C982 235 974 171 1000 171"/>
</svg>

<div class="two-col compact">
<div>
<h3>Реализовано</h3>
<ul>
<li>несколько чат-сессий</li>
<li>автозаголовок и переименование</li>
<li>удаление сессий</li>
<li>summary как служебная память</li>
</ul>
</div>
<div>
<h3>Важно для курса</h3>
<ul>
<li>управление контекстным окном</li>
<li>отделение UI-истории от prompt-истории</li>
<li>persistence в SQLite</li>
</ul>
</div>
</div>

---

## Digest для ссылок: RAG-контекст вместо пустой карточки

<div class="two-col">
<div>
<h3>Проблема</h3>
<p><code>type=link</code> часто содержал только:</p>
<pre><code class="language-yaml">
title: ...
annotation: ...
source_url: ...
</code></pre>
<p>Для RAG этого мало: есть ссылка, но нет плотного знания.</p>
</div>
<div>
<h3>Решение</h3>
<p>LLM создаёт профильный digest:</p>
<ul>
<li>repository profile</li>
<li>documentation profile</li>
<li>product profile</li>
<li>conceptual digest</li>
<li>brief digest</li>
</ul>
<p>Digest хранится в Markdown и индексируется как обычное знание.</p>
</div>
</div>

---

## Agentic engineering: как велась разработка

<div class="flow-strip">
<div>Idea</div>
<div>OpenSpec proposal</div>
<div>Design</div>
<div>Tasks</div>
<div>Implementation</div>
<div>Validation</div>
</div>

<div class="two-col compact">
<div>
<h3>Агенты получили контракты</h3>
<ul>
<li>спецификации <code>SHALL/MUST</code></li>
<li>проверяемые сценарии</li>
<li>задачи с чек-листом</li>
<li>локальные skills для Go/frontend/errors/tests</li>
</ul>
</div>
<div>
<h3>Результат</h3>
<ul>
<li>меньше “сделай красиво”</li>
<li>больше воспроизводимых изменений</li>
<li>код, тесты и спеки сходятся в одну историю</li>
</ul>
</div>
</div>

---

## Live demo сценарий

<div class="timeline">
<div><b>1.</b> Добавить URL или заметку через UI/Telegram</div>
<div><b>2.</b> Показать созданный Markdown и Git diff</div>
<div><b>3.</b> Запустить/проверить индекс</div>
<div><b>4.</b> Найти через hybrid search с diagnostics</div>
<div><b>5.</b> Спросить по найденным источникам в RAG chat</div>
<div><b>6.</b> Продолжить диалог и показать память сессии</div>
</div>

<div class="takeaway">
Демо должно доказать главное: LLM не “болтает рядом”, а проходит весь цикл знания — ingest → index → retrieve → answer → source.
</div>

---

## Что получилось по итогам курса

<div class="result-grid">
<div><b>LLM tools</b><span>function calling для ingestion и metadata extraction</span></div>
<div><b>RAG</b><span>SQLite embeddings, markdown-aware chunking, sources</span></div>
<div><b>Hybrid search</b><span>FTS/keyword/vector + RRF + query rewrite</span></div>
<div><b>Streaming UX</b><span>SSE chat, stop, sources under response</span></div>
<div><b>Memory</b><span>persistent sessions, summaries, context budget</span></div>
<div><b>Agents</b><span>OpenSpec workflow, MCP endpoint, repo-local skills</span></div>
</div>

---

## Инженерные выводы

<div class="two-col">
<div>
<h3>Что сработало</h3>
<ul>
<li>LLM как оркестратор, а не как хранилище</li>
<li>Git/Markdown как проверяемая база</li>
<li>hybrid retrieval вместо чистого vector search</li>
<li>explicit sources и fallback для доверия</li>
</ul>
</div>
<div>
<h3>Что стало понятнее</h3>
<ul>
<li>RAG начинается с качества данных</li>
<li>retrieval нужно объяснять пользователю</li>
<li>агентам нужны контракты, а не только промпты</li>
<li>offline-first ограничивает дизайн в хорошую сторону</li>
</ul>
</div>
</div>

---

## Следующий шаг

<div class="closing">
<h3>Knowledge DB как локальная память для человека и агентов</h3>
<p>База остаётся читаемой и версионируемой, а LLM-слой добавляет ingestion, retrieval, диалог и automation.</p>
</div>

<div class="link-list">
<span>Репозиторий: <b>github.com/strider2038/knowledge-db</b></span>
<span>Спеки: <b>openspec/specs/</b></span>
<span>Презентация: <b>docs/presentations/knowledge-db-overview.md</b></span>
</div>
