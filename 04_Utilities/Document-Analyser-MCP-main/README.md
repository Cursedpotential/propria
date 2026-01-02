# 🔍 FastMCP Document Analyzer

> A comprehensive document analysis server built with the modern FastMCP framework

[![Python](https://img.shields.io/badge/Python-3.8+-blue.svg)](https://python.org)
[![FastMCP](https://img.shields.io/badge/FastMCP-2.3+-green.svg)](https://gofastmcp.com)
[![License](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

## 📋 Table of Contents

- [🌟 Features](#-features)
- [🚀 Quick Start](#-quick-start)
- [📦 Installation](#-installation)
- [🔧 Usage](#-usage)
- [🛠️ Available Tools](#️-available-tools)
- [📊 Sample Data](#-sample-data)
- [🏗️ Project Structure](#️-project-structure)
- [🔄 API Reference](#-api-reference)
- [🧪 Testing](#-testing)
- [📚 Documentation](#-documentation)
- [🤝 Contributing](#-contributing)

## 🌟 Features

### 📖 **Document Analysis**

- **🎭 Sentiment Analysis**: VADER + TextBlob dual-engine sentiment classification
- **🔑 Keyword Extraction**: TF-IDF and frequency-based keyword identification
- **📚 Readability Scoring**: Multiple metrics (Flesch, Flesch-Kincaid, ARI)
- **📊 Text Statistics**: Word count, sentences, paragraphs, and more

### 🗂️ **Document Management**

- **💾 Persistent Storage**: JSON-based document collection with metadata
- **🔍 Smart Search**: TF-IDF semantic similarity search
- **🏷️ Tag System**: Category and tag-based organization
- **📈 Collection Insights**: Comprehensive statistics and analytics

### 🚀 **FastMCP Advantages**

- **⚡ Simple Setup**: 90% less boilerplate than standard MCP
- **🔒 Type Safety**: Full type validation with Pydantic
- **🎯 Modern API**: Decorator-based tool definitions
- **🌐 Multi-Transport**: STDIO, HTTP, and SSE support

## 🚀 Quick Start

### 1. **Clone and Setup**

```bash
git clone <repository-url>
cd document-analyzer
python -m venv venv
source venv/Scripts/activate  # Windows
# source venv/bin/activate    # macOS/Linux
```

### 2. **Install Dependencies**

```bash
pip install -r requirements.txt
```

### 3. **Initialize NLTK Data**

```bash
python -c "import nltk; nltk.download('punkt'); nltk.download('vader_lexicon'); nltk.download('stopwords'); nltk.download('punkt_tab')"
```

### 4. **Run the Server**

```bash
python fastmcp_document_analyzer.py
```

### 5. **Test Everything**

```bash
python test_fastmcp_analyzer.py
```

## 📦 Installation

### **System Requirements**

- Python 3.8 or higher
- 500MB free disk space
- Internet connection (for initial NLTK data download)

### **Dependencies**

```txt
fastmcp>=2.3.0      # Modern MCP framework
textblob>=0.17.1    # Sentiment analysis
nltk>=3.8.1         # Natural language processing
textstat>=0.7.3     # Readability metrics
scikit-learn>=1.3.0 # Machine learning utilities
numpy>=1.24.0       # Numerical computing
pandas>=2.0.0       # Data manipulation
python-dateutil>=2.8.2  # Date handling
```

### **Optional: Virtual Environment**

```bash
# Create virtual environment
python -m venv venv

# Activate (Windows)
venv\Scripts\activate

# Activate (macOS/Linux)
source venv/bin/activate

# Install dependencies
pip install -r requirements.txt
```

## 🔧 Usage

### **Starting the Server**

#### Default (STDIO Transport)

```bash
python fastmcp_document_analyzer.py
```

#### HTTP Transport (for web services)

```bash
python fastmcp_document_analyzer.py --transport http --port 9000
```

#### With Custom Host

```bash
python fastmcp_document_analyzer.py --transport http --host 0.0.0.0 --port 8080
```

### **Basic Usage Examples**

```python
# Analyze a document
result = analyze_document("doc_001")
print(f"Sentiment: {result['sentiment_analysis']['overall_sentiment']}")

# Extract keywords
keywords = extract_keywords("Artificial intelligence is transforming healthcare", 5)
print([kw['keyword'] for kw in keywords])

# Search documents
results = search_documents("machine learning", 3)
print(f"Found {len(results)} relevant documents")

# Get collection statistics
stats = get_collection_stats()
print(f"Total documents: {stats['total_documents']}")
```

## 🛠️ Available Tools

### **Core Analysis Tools**

| Tool                    | Description                   | Example                         |
| ----------------------- | ----------------------------- | ------------------------------- |
| `analyze_document`      | 🔍 Complete document analysis | `analyze_document("doc_001")`   |
| `get_sentiment`         | 😊 Sentiment analysis         | `get_sentiment("I love this!")` |
| `extract_keywords`      | 🔑 Keyword extraction         | `extract_keywords(text, 10)`    |
| `calculate_readability` | 📖 Readability metrics        | `calculate_readability(text)`   |

### **Document Management Tools**

| Tool              | Description           | Example                                  |
| ----------------- | --------------------- | ---------------------------------------- |
| `add_document`    | 📝 Add new document   | `add_document("id", "title", "content")` |
| `get_document`    | 📄 Retrieve document  | `get_document("doc_001")`                |
| `delete_document` | 🗑️ Delete document    | `delete_document("old_doc")`             |
| `list_documents`  | 📋 List all documents | `list_documents("Technology")`           |

### **Search and Discovery Tools**

| Tool                   | Description              | Example                          |
| ---------------------- | ------------------------ | -------------------------------- |
| `search_documents`     | 🔍 Semantic search       | `search_documents("AI", 5)`      |
| `search_by_tags`       | 🏷️ Tag-based search      | `search_by_tags(["AI", "tech"])` |
| `get_collection_stats` | 📊 Collection statistics | `get_collection_stats()`         |

## 📊 Sample Data

The server comes pre-loaded with **16 diverse documents** covering:

| Category        | Documents | Topics                                            |
| --------------- | --------- | ------------------------------------------------- |
| **Technology**  | 4         | AI, Quantum Computing, Privacy, Blockchain        |
| **Science**     | 3         | Space Exploration, Healthcare, Ocean Conservation |
| **Environment** | 2         | Climate Change, Sustainable Agriculture           |
| **Society**     | 3         | Remote Work, Mental Health, Transportation        |
| **Business**    | 2         | Economics, Digital Privacy                        |
| **Culture**     | 2         | Art History, Wellness                             |

### **Sample Document Structure**

```json
{
  "id": "doc_001",
  "title": "The Future of Artificial Intelligence",
  "content": "Artificial intelligence is rapidly transforming...",
  "author": "Dr. Sarah Chen",
  "category": "Technology",
  "tags": ["AI", "technology", "future", "ethics"],
  "language": "en",
  "created_at": "2024-01-15T10:30:00"
}
```

## 🏗️ Project Structure

```
document-analyzer/
├── 📁 analyzer/                    # Core analysis engine
│   ├── __init__.py
│   └── document_analyzer.py       # Sentiment, keywords, readability
├── 📁 storage/                     # Document storage system
│   ├── __init__.py
│   └── document_storage.py        # JSON storage, search, management
├── 📁 data/                        # Sample data
│   ├── __init__.py
│   └── sample_documents.py        # 16 sample documents
├── 📄 fastmcp_document_analyzer.py # 🌟 Main FastMCP server
├── 📄 test_fastmcp_analyzer.py    # Comprehensive test suite
├── 📄 requirements.txt            # Python dependencies
├── 📄 documents.json              # Persistent document storage
├── 📄 README.md                   # This documentation
├── 📄 FASTMCP_COMPARISON.md       # FastMCP vs Standard MCP
├── 📄 .gitignore                  # Git ignore patterns
└── 📁 venv/                       # Virtual environment (optional)
```

## 🔄 API Reference

### **Document Analysis**

#### `analyze_document(document_id: str) -> Dict[str, Any]`

Performs comprehensive analysis of a document.

**Parameters:**

- `document_id` (str): Unique document identifier

**Returns:**

```json
{
  "document_id": "doc_001",
  "title": "Document Title",
  "sentiment_analysis": {
    "overall_sentiment": "positive",
    "confidence": 0.85,
    "vader_scores": {...},
    "textblob_scores": {...}
  },
  "keywords": [
    {"keyword": "artificial", "frequency": 5, "relevance_score": 2.3}
  ],
  "readability": {
    "flesch_reading_ease": 45.2,
    "reading_level": "Difficult",
    "grade_level": "Grade 12"
  },
  "basic_statistics": {
    "word_count": 119,
    "sentence_count": 8,
    "paragraph_count": 1
  }
}
```

#### `get_sentiment(text: str) -> Dict[str, Any]`

Analyzes sentiment of any text.

**Parameters:**

- `text` (str): Text to analyze

**Returns:**

```json
{
  "overall_sentiment": "positive",
  "confidence": 0.85,
  "vader_scores": {
    "compound": 0.7269,
    "positive": 0.294,
    "negative": 0.0,
    "neutral": 0.706
  },
  "textblob_scores": {
    "polarity": 0.5,
    "subjectivity": 0.6
  }
}
```

### **Document Management**

#### `add_document(...) -> Dict[str, str]`

Adds a new document to the collection.

**Parameters:**

- `id` (str): Unique document ID
- `title` (str): Document title
- `content` (str): Document content
- `author` (str, optional): Author name
- `category` (str, optional): Document category
- `tags` (List[str], optional): Tags list
- `language` (str, optional): Language code

**Returns:**

```json
{
  "status": "success",
  "message": "Document 'my_doc' added successfully",
  "document_count": 17
}
```

### **Search and Discovery**

#### `search_documents(query: str, limit: int = 10) -> List[Dict[str, Any]]`

Performs semantic search across documents.

**Parameters:**

- `query` (str): Search query
- `limit` (int): Maximum results

**Returns:**

```json
[
  {
    "id": "doc_001",
    "title": "AI Document",
    "similarity_score": 0.8542,
    "content_preview": "First 200 characters...",
    "tags": ["AI", "technology"]
  }
]
```

## 🧪 Testing

### **Run All Tests**

```bash
python test_fastmcp_analyzer.py
```

### **Test Categories**

- ✅ **Server Initialization**: FastMCP server setup
- ✅ **Sentiment Analysis**: VADER and TextBlob integration
- ✅ **Keyword Extraction**: TF-IDF and frequency analysis
- ✅ **Readability Calculation**: Multiple readability metrics
- ✅ **Document Analysis**: Full document processing
- ✅ **Document Search**: Semantic similarity search
- ✅ **Collection Statistics**: Analytics and insights
- ✅ **Document Management**: CRUD operations
- ✅ **Tag Search**: Tag-based filtering

### **Expected Test Output**

```
=== Testing FastMCP Document Analyzer ===

✓ FastMCP server module imported successfully
✓ Server initialized successfully
✓ Sentiment analysis working
✓ Keyword extraction working
✓ Readability calculation working
✓ Document analysis working
✓ Document search working
✓ Collection statistics working
✓ Document listing working
✓ Document addition and deletion working
✓ Tag search working

=== All FastMCP tests completed successfully! ===
```

## 📚 Documentation

### **Additional Resources**

- 📖 [FastMCP Documentation](https://gofastmcp.com)
- 📖 [MCP Protocol Specification](https://modelcontextprotocol.io)
- 📖 [FASTMCP_COMPARISON.md](FASTMCP_COMPARISON.md) - FastMCP vs Standard MCP

### **Key Concepts**

#### **Sentiment Analysis**

Uses dual-engine approach:

- **VADER**: Rule-based, excellent for social media text
- **TextBlob**: Machine learning-based, good for general text

#### **Keyword Extraction**

Combines multiple approaches:

- **TF-IDF**: Term frequency-inverse document frequency
- **Frequency Analysis**: Simple word frequency counting
- **Relevance Scoring**: Weighted combination of both methods

#### **Readability Metrics**

Provides multiple readability scores:

- **Flesch Reading Ease**: 0-100 scale (higher = easier)
- **Flesch-Kincaid Grade**: US grade level
- **ARI**: Automated Readability Index

#### **Document Search**

Uses TF-IDF vectorization with cosine similarity:

- Converts documents to numerical vectors
- Calculates similarity between query and documents
- Returns ranked results with similarity scores

## 🤝 Contributing

### **Development Setup**

```bash
# Clone repository
git clone <repository-url>
cd document-analyzer

# Create development environment
python -m venv venv
source venv/Scripts/activate  # Windows
pip install -r requirements.txt

# Run tests
python test_fastmcp_analyzer.py
```

### **Adding New Tools**

FastMCP makes it easy to add new tools:

```python
@mcp.tool
def my_new_tool(param: str) -> Dict[str, Any]:
    """
    🔧 Description of what this tool does.

    Args:
        param: Parameter description

    Returns:
        Return value description
    """
    # Implementation here
    return {"result": "success"}
```

### **Code Style**

- Use type hints for all functions
- Add comprehensive docstrings
- Include error handling
- Follow PEP 8 style guidelines
- Add emoji icons for better readability

### **Testing New Features**

1. Add your tool to the main server file
2. Create test cases in the test file
3. Run the test suite to ensure everything works
4. Update documentation as needed

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- **FastMCP Team** for the excellent framework
- **NLTK Team** for natural language processing tools
- **TextBlob Team** for sentiment analysis capabilities
- **Scikit-learn Team** for machine learning utilities

---

**Made with ❤️ using FastMCP**

> 🚀 Ready to analyze documents? Start with `python fastmcp_document_analyzer.py`
