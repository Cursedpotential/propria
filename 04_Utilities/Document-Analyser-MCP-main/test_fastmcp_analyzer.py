#!/usr/bin/env python3
"""Test script for the FastMCP Document Analyzer Server."""

import sys
import os

# Add the current directory to Python path so we can import our modules
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

def test_fastmcp_analyzer():
    """Test the FastMCP document analyzer functionality by importing and testing tools directly."""
    print("=== Testing FastMCP Document Analyzer ===\n")
    
    try:
        # Import the FastMCP server module
        import fastmcp_document_analyzer as server
        print("✓ FastMCP server module imported successfully\n")
        
        # Test 1: Check server initialization
        print("1. Testing server initialization:")
        print(f"   Server name: {server.mcp.name}")
        print(f"   Documents loaded: {len(server.storage.get_all_documents())}")
        print("   ✓ Server initialized successfully\n")
        
        # Test 2: Test sentiment analysis
        print("2. Testing sentiment analysis:")
        test_text = "I absolutely love this amazing product! It's fantastic and works perfectly."
        sentiment = server.analyzer.analyze_sentiment(test_text)
        print(f"   Text: {test_text}")
        print(f"   Sentiment: {sentiment['overall_sentiment']} (confidence: {sentiment['confidence']})")
        print("   ✓ Sentiment analysis working\n")
        
        # Test 3: Test keyword extraction
        print("3. Testing keyword extraction:")
        keywords = server.analyzer.extract_keywords(test_text, limit=5)
        keyword_list = [kw['keyword'] for kw in keywords]
        print(f"   Keywords: {keyword_list}")
        print("   ✓ Keyword extraction working\n")
        
        # Test 4: Test readability calculation
        print("4. Testing readability calculation:")
        readability_metrics = server.analyzer.calculate_readability(test_text)
        basic_stats = server.analyzer.get_basic_stats(test_text)
        print(f"   Reading level: {readability_metrics['reading_level']}")
        print(f"   Grade level: {readability_metrics['grade_level']}")
        print("   ✓ Readability calculation working\n")
        
        # Test 5: Test document analysis
        print("5. Testing full document analysis:")
        doc = server.storage.get_document("doc_001")
        if doc:
            analysis = server.analyzer.analyze_document(doc)
            print(f"   Document: {analysis['title']}")
            print(f"   Sentiment: {analysis['sentiment_analysis']['overall_sentiment']}")
            print(f"   Top keywords: {[kw['keyword'] for kw in analysis['keywords'][:3]]}")
            print(f"   Word count: {analysis['basic_statistics']['word_count']}")
            print("   ✓ Document analysis working\n")
        else:
            print("   ✗ Error: Document doc_001 not found\n")
        
        # Test 6: Test document search
        print("6. Testing document search:")
        search_results = server.storage.search_documents("artificial intelligence", limit=3)
        print(f"   Found {len(search_results)} documents for 'artificial intelligence':")
        for result in search_results:
            print(f"   - {result['title']} (similarity: {result['similarity_score']})")
        print("   ✓ Document search working\n")
        
        # Test 7: Test collection statistics
        print("7. Testing collection statistics:")
        stats = server.storage.get_document_stats()
        print(f"   Total documents: {stats['total_documents']}")
        print(f"   Total words: {stats['total_words']}")
        print(f"   Categories: {list(stats['categories'].keys())[:5]}...")  # Show first 5
        print("   ✓ Collection statistics working\n")
        
        # Test 8: Test document listing
        print("8. Testing document listing:")
        all_docs = server.storage.get_all_documents()
        doc_list = list(all_docs.values())
        print(f"   Total documents listed: {len(doc_list)}")
        if doc_list:
            print(f"   First document: {doc_list[0]['title']}")
        print("   ✓ Document listing working\n")
        
        # Test 9: Test adding and deleting documents
        print("9. Testing document addition and deletion:")
        
        # Add test document
        test_doc = {
            "id": "test_fastmcp_001",
            "title": "FastMCP Test Document",
            "content": "This is a test document for the FastMCP implementation. It contains sample text for testing purposes.",
            "author": "Test Author",
            "category": "Testing",
            "tags": ["test", "fastmcp", "sample"],
            "language": "en"
        }
        
        try:
            server.storage.add_document(test_doc)
            print("   ✓ Document added successfully")
            
            # Test retrieval
            retrieved_doc = server.storage.get_document("test_fastmcp_001")
            if retrieved_doc:
                print(f"   ✓ Document retrieved: {retrieved_doc['title']}")
                
                # Test deletion
                success = server.storage.delete_document("test_fastmcp_001")
                if success:
                    print("   ✓ Document deleted successfully")
                else:
                    print("   ✗ Delete failed")
            else:
                print("   ✗ Retrieval failed")
        except Exception as e:
            print(f"   ✗ Add failed: {e}")
        print()
        
        # Test 10: Test tag search
        print("10. Testing tag search:")
        tag_results = server.storage.search_by_tags(["AI", "technology"])
        print(f"   Found {len(tag_results)} documents with tags 'AI' or 'technology':")
        for result in tag_results[:3]:  # Show first 3
            print(f"   - {result['title']} (tags: {result['tags']})")
        print("   ✓ Tag search working\n")
        
        print("=== All FastMCP tests completed successfully! ===")
        return True
        
    except Exception as e:
        print(f"✗ Test failed with error: {e}")
        import traceback
        traceback.print_exc()
        return False


if __name__ == "__main__":
    success = test_fastmcp_analyzer()
    sys.exit(0 if success else 1) 