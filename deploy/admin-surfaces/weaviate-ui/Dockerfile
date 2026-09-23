# Weaviate UI: the community "Weaviate Cluster" Streamlit app (MIT, Mohamed Shahin, Weaviate support
# engineer), pinned to one commit. Weaviate has no official UI for self-hosted instances (Weaviate
# staff, forum 2025-05-08).
# Byline: Claude Code · Opus 5.5 · 2026-09-23. Owner 08:15: "find a weaviate management surface,
# what we route to on the homepage portal is bullshit".
# Built on ovh-files: docker build -t propria/weaviate-ui:389a3f9 deploy/admin-surfaces/weaviate-ui
FROM python:3.12-slim
ARG REF=389a3f9d0fc452e232e741d55db9ee2c69f84b85
RUN apt-get update && apt-get install -y --no-install-recommends git && apt-get clean
RUN git clone https://github.com/Shah91n/weaviate-cluster-webapp /app && git -C /app checkout "$REF"
WORKDIR /app
RUN pip install --no-cache-dir -r requirements.txt
EXPOSE 8501
CMD ["streamlit", "run", "streamlit_app.py", "--server.port", "8501", "--server.address", "0.0.0.0", "--server.headless", "true", "--browser.gatherUsageStats", "false"]
