#!/usr/bin/env python3
"""
FastMCP Document Analyzer Server

A comprehensive document analysis server built with FastMCP framework.
Provides sentiment analysis, keyword extraction, readability scoring,
and document search capabilities.

Author: AI Assistant
Version: 1.0.0
"""

from typing import Dict, List, Any, Optional
from fastmcp import FastMCP

# Import our custom modules
from analyzer.document_analyzer import DocumentAnalyzer
from storage.document_storage import DocumentStorage

# =============================================================================
# SERVER INITIALIZATION
# =============================================================================

# Initialize the FastMCP server with configuration
mcp = FastMCP(
    name="document-analyzer",
    on_duplicate_tools="error"  # Prevent duplicate tool registration
)

# Initialize core components
print("🔧 Initializing Document Analyzer components...")
analyzer = DocumentAnalyzer()
storage = DocumentStorage()

# Display initialization status
document_count = len(storage.get_all_documents())
print(f"✅ FastMCP Document Analyzer initialized successfully!")
print(f"📚 Loaded {document_count} documents from storage")
print(f"🔍 Ready to analyze documents with {len(mcp._tools) if hasattr(mcp, '_tools') else 'multiple'} tools")

# =============================================================================
# CORE ANALYSIS TOOLS
# =============================================================================

@mcp.tool
def analyze_document(document_id: str) -> Dict[str, Any]:
    """
    🔍 Perform comprehensive analysis of a document.
    
    Analyzes a document for sentiment, keywords, readability, and basic statistics.
    This is the main analysis function that combines all analysis features.
    
    Args:
        document_id: Unique identifier of the document to analyze
        
    Returns:
        Complete analysis including:
        - Sentiment analysis (positive/negative/neutral)
        - Top keywords with relevance scores
        - Readability metrics and grade level
        - Basic statistics (word count, sentences, etc.)
        
    Example:
        analyze_document("doc_001") -> Complete analysis of document doc_001
    """
    # Retrieve document from storage
    document = storage.get_document(document_id)
    if not document:
        return {
            "error": f"Document with ID '{document_id}' not found",
            "available_documents": list(storage.get_all_documents().keys())[:5]
        }
    
    # Perform comprehensive analysis
    try:
        analysis_result = analyzer.analyze_document(document)
        analysis_result["analysis_timestamp"] = "now"
        return analysis_result
    except Exception as e:
        return {"error": f"Analysis failed: {str(e)}"}


@mcp.tool
def get_sentiment(text: str) -> Dict[str, Any]:
    """
    😊 Analyze sentiment of any text input.
    
    Uses both VADER and TextBlob sentiment analysis algorithms
    to provide comprehensive sentiment classification.
    
    Args:
        text: Text content to analyze for sentiment
        
    Returns:
        Sentiment analysis including:
        - Overall sentiment (positive/negative/neutral)
        - Confidence score
        - Detailed VADER and TextBlob scores
        
    Example:
        get_sentiment("I love this product!") -> {"overall_sentiment": "positive", ...}
    """
    if not text or not text.strip():
        return {"error": "Text cannot be empty"}
    
    try:
        return analyzer.analyze_sentiment(text)
    except Exception as e:
        return {"error": f"Sentiment analysis failed: {str(e)}"}


@mcp.tool
def extract_keywords(text: str, limit: int = 10) -> List[Dict[str, Any]]:
    """
    🔑 Extract top keywords from text using advanced algorithms.
    
    Combines TF-IDF and frequency analysis to identify the most
    relevant keywords in the provided text.
    
    Args:
        text: Text content to extract keywords from
        limit: Maximum number of keywords to return (default: 10)
        
    Returns:
        List of keywords with:
        - Keyword text
        - Frequency count
        - Relevance score
        
    Example:
        extract_keywords("AI and machine learning", 5) -> [{"keyword": "learning", ...}, ...]
    """
    if not text or not text.strip():
        return []
    
    if limit < 1:
        limit = 10
    elif limit > 50:  # Reasonable upper limit
        limit = 50
    
    try:
        return analyzer.extract_keywords(text, limit)
    except Exception as e:
        return [{"error": f"Keyword extraction failed: {str(e)}"}]


@mcp.tool
def calculate_readability(text: str) -> Dict[str, Any]:
    """
    📖 Calculate readability metrics for any text.
    
    Provides multiple readability scores including Flesch Reading Ease,
    Flesch-Kincaid Grade Level, and Automated Readability Index.
    
    Args:
        text: Text content to analyze for readability
        
    Returns:
        Readability metrics and basic text statistics:
        - Reading level (Easy/Average/Difficult)
        - Grade level
        - Various readability scores
        - Word count, sentence count, etc.
        
    Example:
        calculate_readability("Simple text here.") -> {"reading_level": "Easy", ...}
    """
    if not text or not text.strip():
        return {"error": "Text cannot be empty"}
    
    try:
        readability_metrics = analyzer.calculate_readability(text)
        basic_stats = analyzer.get_basic_stats(text)
        
        return {
            "readability_metrics": readability_metrics,
            "basic_statistics": basic_stats,
            "text_length": len(text),
            "analysis_summary": f"Grade {readability_metrics.get('grade_level', 'Unknown')} - {readability_metrics.get('reading_level', 'Unknown')} reading level"
        }
    except Exception as e:
        return {"error": f"Readability analysis failed: {str(e)}"}

# =============================================================================
# DOCUMENT MANAGEMENT TOOLS
# =============================================================================

@mcp.tool
def add_document(
    id: str,
    title: str, 
    content: str,
    author: str = "Unknown",
    category: str = "General",
    tags: List[str] = None,
    language: str = "en"
) -> Dict[str, str]:
    """
    📝 Add a new document to the collection.
    
    Creates a new document with metadata and stores it in the collection.
    The document will be available for analysis and search immediately.
    
    Args:
        id: Unique document identifier (required)
        title: Document title (required)
        content: Document content/text (required)
        author: Document author (optional, default: "Unknown")
        category: Document category (optional, default: "General")
        tags: List of tags for categorization (optional)
        language: Document language code (optional, default: "en")
        
    Returns:
        Success confirmation or error message
        
    Example:
        add_document("my_doc", "My Title", "Content here", tags=["important"])
    """
    # Validate required fields
    if not id or not id.strip():
        return {"status": "error", "message": "Document ID cannot be empty"}
    if not title or not title.strip():
        return {"status": "error", "message": "Document title cannot be empty"}
    if not content or not content.strip():
        return {"status": "error", "message": "Document content cannot be empty"}
    
    # Set default tags if None
    if tags is None:
        tags = []
    
    # Create document data structure
    document_data = {
        "id": id.strip(),
        "title": title.strip(),
        "content": content.strip(),
        "author": author.strip() if author else "Unknown",
        "category": category.strip() if category else "General",
        "tags": [tag.strip() for tag in tags if tag.strip()],
        "language": language.strip() if language else "en"
    }
    
    try:
        storage.add_document(document_data)
        return {
            "status": "success", 
            "message": f"Document '{id}' added successfully",
            "document_count": len(storage.get_all_documents())
        }
    except Exception as e:
        return {"status": "error", "message": f"Failed to add document: {str(e)}"}


@mcp.tool
def get_document(document_id: str) -> Dict[str, Any]:
    """
    📄 Retrieve a specific document by ID.
    
    Fetches complete document data including content and metadata.
    
    Args:
        document_id: Unique identifier of the document to retrieve
        
    Returns:
        Complete document data or error message
        
    Example:
        get_document("doc_001") -> {"id": "doc_001", "title": "...", ...}
    """
    if not document_id or not document_id.strip():
        return {"error": "Document ID cannot be empty"}
    
    document = storage.get_document(document_id.strip())
    if not document:
        available_docs = list(storage.get_all_documents().keys())
        return {
            "error": f"Document with ID '{document_id}' not found",
            "available_documents": available_docs[:10],  # Show first 10
            "total_documents": len(available_docs)
        }
    
    # Convert datetime to string for JSON serialization
    display_doc = document.copy()
    if 'created_at' in display_doc and hasattr(display_doc['created_at'], 'isoformat'):
        display_doc['created_at'] = display_doc['created_at'].isoformat()
    
    return display_doc


@mcp.tool
def delete_document(document_id: str) -> Dict[str, str]:
    """
    🗑️ Delete a document from the collection.
    
    Permanently removes a document and all its associated data.
    This action cannot be undone.
    
    Args:
        document_id: Unique identifier of the document to delete
        
    Returns:
        Success confirmation or error message
        
    Example:
        delete_document("old_doc") -> {"status": "success", ...}
    """
    if not document_id or not document_id.strip():
        return {"status": "error", "message": "Document ID cannot be empty"}
    
    success = storage.delete_document(document_id.strip())
    if success:
        return {
            "status": "success", 
            "message": f"Document '{document_id}' deleted successfully",
            "remaining_documents": len(storage.get_all_documents())
        }
    else:
        return {
            "status": "error", 
            "message": f"Document with ID '{document_id}' not found or could not be deleted"
        }

# =============================================================================
# SEARCH AND DISCOVERY TOOLS
# =============================================================================

@mcp.tool
def search_documents(query: str, limit: int = 10) -> List[Dict[str, Any]]:
    """
    🔍 Search documents using semantic similarity.
    
    Performs intelligent content-based search using TF-IDF and cosine similarity
    to find documents most relevant to your query.
    
    Args:
        query: Search query text
        limit: Maximum number of results to return (default: 10)
        
    Returns:
        List of matching documents with:
        - Document metadata
        - Similarity score
        - Content preview
        
    Example:
        search_documents("artificial intelligence", 5) -> [{"title": "AI Doc", ...}, ...]
    """
    if not query or not query.strip():
        return []
    
    if limit < 1:
        limit = 10
    elif limit > 50:  # Reasonable upper limit
        limit = 50
    
    try:
        results = storage.search_documents(query.strip(), limit)
        
        # Format results for better display
        formatted_results = []
        for doc in results:
            content = doc.get("content", "")
            formatted_results.append({
                "id": doc.get("id"),
                "title": doc.get("title"),
                "author": doc.get("author"),
                "category": doc.get("category"),
                "similarity_score": round(doc.get("similarity_score", 0), 4),
                "content_preview": content[:200] + "..." if len(content) > 200 else content,
                "tags": doc.get("tags", []),
                "word_count": len(content.split()) if content else 0
            })
        
        return formatted_results
    except Exception as e:
        return [{"error": f"Search failed: {str(e)}"}]


@mcp.tool
def search_by_tags(tags: List[str]) -> List[Dict[str, Any]]:
    """
    🏷️ Search documents by tags.
    
    Finds all documents that contain any of the specified tags.
    Useful for finding documents in specific categories or topics.
    
    Args:
        tags: List of tags to search for
        
    Returns:
        List of documents containing any of the specified tags
        
    Example:
        search_by_tags(["AI", "technology"]) -> [{"title": "AI Doc", ...}, ...]
    """
    if not tags or not any(tag.strip() for tag in tags):
        return []
    
    # Clean up tags
    clean_tags = [tag.strip() for tag in tags if tag.strip()]
    
    try:
        results = storage.search_by_tags(clean_tags)
        
        # Format results for better display
        formatted_results = []
        for doc in results:
            content = doc.get("content", "")
            formatted_results.append({
                "id": doc.get("id"),
                "title": doc.get("title"),
                "author": doc.get("author"),
                "category": doc.get("category"),
                "tags": doc.get("tags", []),
                "matching_tags": [tag for tag in doc.get("tags", []) if tag in clean_tags],
                "word_count": len(content.split()) if content else 0
            })
        
        return formatted_results
    except Exception as e:
        return [{"error": f"Tag search failed: {str(e)}"}]


@mcp.tool
def list_documents(category: Optional[str] = None) -> List[Dict[str, Any]]:
    """
    📋 List all documents with metadata.
    
    Provides an overview of all documents in the collection,
    optionally filtered by category.
    
    Args:
        category: Optional category filter (e.g., "Technology", "Science")
        
    Returns:
        List of documents with basic metadata:
        - ID, title, author, category
        - Tags, word count, creation date
        
    Example:
        list_documents() -> [{"id": "doc_001", ...}, ...]
        list_documents("Technology") -> [filtered results]
    """
    try:
        if category and category.strip():
            documents = storage.search_by_category(category.strip())
        else:
            documents = list(storage.get_all_documents().values())
        
        # Create summary list with essential information
        doc_list = []
        for doc in documents:
            content = doc.get("content", "")
            created_at = doc.get("created_at", "")
            
            doc_list.append({
                "id": doc.get("id"),
                "title": doc.get("title"),
                "author": doc.get("author"),
                "category": doc.get("category"),
                "tags": doc.get("tags", []),
                "word_count": len(content.split()) if content else 0,
                "character_count": len(content) if content else 0,
                "created_at": created_at.isoformat() if hasattr(created_at, 'isoformat') else str(created_at),
                "language": doc.get("language", "en")
            })
        
        return doc_list
    except Exception as e:
        return [{"error": f"Failed to list documents: {str(e)}"}]

# =============================================================================
# COLLECTION STATISTICS AND INSIGHTS
# =============================================================================

@mcp.tool
def get_collection_stats() -> Dict[str, Any]:
    """
    📊 Get comprehensive statistics about the document collection.
    
    Provides insights into the document collection including counts,
    categories, authors, and other useful metrics.
    
    Returns:
        Collection statistics including:
        - Total documents and words
        - Category distribution
        - Author distribution
        - Tag frequency
        - Average document length
        
    Example:
        get_collection_stats() -> {"total_documents": 16, "categories": {...}, ...}
    """
    try:
        stats = storage.get_document_stats()
        
        # Add some additional insights
        if stats.get("total_documents", 0) > 0:
            avg_words = stats.get("total_words", 0) / stats.get("total_documents", 1)
            stats["average_words_per_document"] = round(avg_words, 1)
            
            # Add category insights
            categories = stats.get("categories", {})
            if categories:
                stats["most_common_category"] = max(categories.items(), key=lambda x: x[1])[0]
                stats["category_diversity"] = len(categories)
        
        return stats
    except Exception as e:
        return {"error": f"Failed to get collection stats: {str(e)}"}

# =============================================================================
# SERVER STARTUP
# =============================================================================

def main():
    """
    🚀 Main server startup function.
    
    Initializes and runs the FastMCP server with proper error handling.
    """
    print("\n" + "="*60)
    print("🚀 Starting FastMCP Document Analyzer Server")
    print("="*60)
    print(f"📚 Collection: {len(storage.get_all_documents())} documents loaded")
    print(f"🔧 Tools: {len([name for name in dir(mcp) if not name.startswith('_')])} analysis tools available")
    print("🌐 Transport: STDIO (default) - ready for MCP clients")
    print("📖 Documentation: See README.md for usage instructions")
    print("="*60)
    
    try:
        # Run the FastMCP server
        mcp.run()
    except KeyboardInterrupt:
        print("\n👋 Server shutdown requested by user")
    except Exception as e:
        print(f"\n❌ Server error: {e}")
        raise


if __name__ == "__main__":
    main() 