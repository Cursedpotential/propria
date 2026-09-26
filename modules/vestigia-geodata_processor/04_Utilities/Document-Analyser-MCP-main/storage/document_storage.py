#!/usr/bin/env python3
"""Document storage and search functionality."""

import json
import os
from typing import Dict, List, Any, Optional
from datetime import datetime
import pandas as pd
from sklearn.feature_extraction.text import TfidfVectorizer
from sklearn.metrics.pairwise import cosine_similarity

from data.sample_documents import SAMPLE_DOCUMENTS


class DocumentStorage:
    """Document storage with search and management capabilities."""
    
    def __init__(self, storage_file: str = "documents.json"):
        """
        Initialize document storage.
        
        Args:
            storage_file: Path to JSON file for persistent storage
        """
        self.storage_file = storage_file
        self.documents = {}
        self.tfidf_vectorizer = None
        self.tfidf_matrix = None
        
        # Load existing documents or initialize with sample data
        self._load_documents()
        self._update_search_index()
    
    def _load_documents(self):
        """Load documents from storage file or initialize with sample data."""
        if os.path.exists(self.storage_file):
            try:
                with open(self.storage_file, 'r', encoding='utf-8') as f:
                    data = json.load(f)
                    # Convert datetime strings back to datetime objects
                    for doc_id, doc in data.items():
                        if 'created_at' in doc and isinstance(doc['created_at'], str):
                            try:
                                doc['created_at'] = datetime.fromisoformat(doc['created_at'])
                            except:
                                pass
                    self.documents = data
                print(f"Loaded {len(self.documents)} documents from {self.storage_file}")
            except Exception as e:
                print(f"Error loading documents: {e}")
                self._initialize_with_sample_data()
        else:
            self._initialize_with_sample_data()
    
    def _initialize_with_sample_data(self):
        """Initialize storage with sample documents."""
        for doc in SAMPLE_DOCUMENTS:
            self.documents[doc['id']] = doc
        self._save_documents()
        print(f"Initialized with {len(SAMPLE_DOCUMENTS)} sample documents")
    
    def _save_documents(self):
        """Save documents to storage file."""
        try:
            # Convert datetime objects to strings for JSON serialization
            serializable_docs = {}
            for doc_id, doc in self.documents.items():
                serializable_doc = doc.copy()
                if 'created_at' in serializable_doc and hasattr(serializable_doc['created_at'], 'isoformat'):
                    serializable_doc['created_at'] = serializable_doc['created_at'].isoformat()
                serializable_docs[doc_id] = serializable_doc
            
            with open(self.storage_file, 'w', encoding='utf-8') as f:
                json.dump(serializable_docs, f, indent=2, ensure_ascii=False)
        except Exception as e:
            print(f"Error saving documents: {e}")
    
    def _update_search_index(self):
        """Update TF-IDF search index for document search."""
        if not self.documents:
            return
        
        # Prepare documents for TF-IDF
        doc_texts = []
        doc_ids = []
        
        for doc_id, doc in self.documents.items():
            # Combine title and content for search
            text = f"{doc.get('title', '')} {doc.get('content', '')}"
            doc_texts.append(text)
            doc_ids.append(doc_id)
        
        # Create TF-IDF vectorizer and matrix
        self.tfidf_vectorizer = TfidfVectorizer(
            stop_words='english',
            max_features=5000,
            ngram_range=(1, 2)
        )
        
        try:
            self.tfidf_matrix = self.tfidf_vectorizer.fit_transform(doc_texts)
            self.doc_ids = doc_ids
        except Exception as e:
            print(f"Error updating search index: {e}")
            self.tfidf_matrix = None
    
    def add_document(self, document_data: Dict[str, Any]) -> bool:
        """
        Add a new document to storage.
        
        Args:
            document_data: Document dictionary with required fields
            
        Returns:
            True if successful, False otherwise
        """
        required_fields = ['id', 'title', 'content']
        
        # Validate required fields
        for field in required_fields:
            if field not in document_data:
                raise ValueError(f"Missing required field: {field}")
        
        # Check if document ID already exists
        if document_data['id'] in self.documents:
            raise ValueError(f"Document with ID {document_data['id']} already exists")
        
        # Add metadata if not provided
        if 'created_at' not in document_data:
            document_data['created_at'] = datetime.now()
        
        if 'author' not in document_data:
            document_data['author'] = "Unknown"
        
        if 'category' not in document_data:
            document_data['category'] = "General"
        
        if 'tags' not in document_data:
            document_data['tags'] = []
        
        if 'language' not in document_data:
            document_data['language'] = "en"
        
        # Add document
        self.documents[document_data['id']] = document_data
        
        # Save and update search index
        self._save_documents()
        self._update_search_index()
        
        return True
    
    def get_document(self, document_id: str) -> Optional[Dict[str, Any]]:
        """
        Retrieve a document by ID.
        
        Args:
            document_id: Document ID
            
        Returns:
            Document dictionary or None if not found
        """
        return self.documents.get(document_id)
    
    def get_all_documents(self) -> Dict[str, Dict[str, Any]]:
        """
        Get all documents.
        
        Returns:
            Dictionary of all documents
        """
        return self.documents.copy()
    
    def search_documents(self, query: str, limit: int = 10) -> List[Dict[str, Any]]:
        """
        Search documents using TF-IDF similarity.
        
        Args:
            query: Search query
            limit: Maximum number of results
            
        Returns:
            List of matching documents with similarity scores
        """
        if self.tfidf_matrix is None or not self.tfidf_vectorizer:
            return []
        
        try:
            # Transform query using the same vectorizer
            query_vector = self.tfidf_vectorizer.transform([query])
            
            # Calculate cosine similarity
            similarities = cosine_similarity(query_vector, self.tfidf_matrix).flatten()
            
            # Get top results
            top_indices = similarities.argsort()[-limit:][::-1]
            
            results = []
            for idx in top_indices:
                if similarities[idx] > 0:  # Only include documents with some similarity
                    doc_id = self.doc_ids[idx]
                    doc = self.documents[doc_id].copy()
                    doc['similarity_score'] = round(similarities[idx], 4)
                    results.append(doc)
            
            return results
            
        except Exception as e:
            print(f"Error searching documents: {e}")
            return []
    
    def search_by_category(self, category: str) -> List[Dict[str, Any]]:
        """
        Search documents by category.
        
        Args:
            category: Category name
            
        Returns:
            List of documents in the category
        """
        results = []
        for doc in self.documents.values():
            if doc.get('category', '').lower() == category.lower():
                results.append(doc)
        return results
    
    def search_by_tags(self, tags: List[str]) -> List[Dict[str, Any]]:
        """
        Search documents by tags.
        
        Args:
            tags: List of tags to search for
            
        Returns:
            List of documents containing any of the tags
        """
        results = []
        search_tags = [tag.lower() for tag in tags]
        
        for doc in self.documents.values():
            doc_tags = [tag.lower() for tag in doc.get('tags', [])]
            if any(tag in doc_tags for tag in search_tags):
                results.append(doc)
        return results
    
    def get_document_stats(self) -> Dict[str, Any]:
        """
        Get statistics about the document collection.
        
        Returns:
            Dictionary with collection statistics
        """
        if not self.documents:
            return {"total_documents": 0}
        
        # Basic stats
        total_docs = len(self.documents)
        
        # Category distribution
        categories = {}
        authors = {}
        languages = {}
        
        total_words = 0
        
        for doc in self.documents.values():
            # Count categories
            category = doc.get('category', 'Unknown')
            categories[category] = categories.get(category, 0) + 1
            
            # Count authors
            author = doc.get('author', 'Unknown')
            authors[author] = authors.get(author, 0) + 1
            
            # Count languages
            language = doc.get('language', 'Unknown')
            languages[language] = languages.get(language, 0) + 1
            
            # Count words (rough estimate)
            content = doc.get('content', '')
            words = len(content.split())
            total_words += words
        
        return {
            "total_documents": total_docs,
            "total_words": total_words,
            "average_words_per_document": round(total_words / total_docs, 2) if total_docs > 0 else 0,
            "categories": dict(sorted(categories.items(), key=lambda x: x[1], reverse=True)),
            "authors": dict(sorted(authors.items(), key=lambda x: x[1], reverse=True)),
            "languages": languages,
            "unique_categories": len(categories),
            "unique_authors": len(authors)
        }
    
    def delete_document(self, document_id: str) -> bool:
        """
        Delete a document by ID.
        
        Args:
            document_id: Document ID to delete
            
        Returns:
            True if successful, False if document not found
        """
        if document_id not in self.documents:
            return False
        
        del self.documents[document_id]
        self._save_documents()
        self._update_search_index()
        return True 