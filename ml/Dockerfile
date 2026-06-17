FROM python:3.10-slim
WORKDIR /app
RUN pip install --no-cache-dir \
    fastapi \
    uvicorn \
    faiss-cpu \
    numpy \
    pandas \
    pennylane
COPY . .
EXPOSE 8001
ENV KMP_DUPLICATE_LIB_OK=TRUE
CMD ["uvicorn", "serve:app", "--host", "0.0.0.0", "--port", "8001"]
