#!/usr/bin/env python3
"""Document analysis module for sentiment, keywords, and readability."""

import re
import string
from typing import Dict, List, Any, Tuple
from collections import Counter
from datetime import datetime

import nltk
from nltk.sentiment import SentimentIntensityAnalyzer
from nltk.corpus import stopwords
from nltk.tokenize import word_tokenize, sent_tokenize
from textblob import TextBlob
from textstat import flesch_reading_ease, flesch_kincaid_grade, automated_readability_index
from sklearn.feature_extraction.text import TfidfVectorizer


class DocumentAnalyzer:
    """Comprehensive document analysis for sentiment, keywords, and readability."""
    
    def __init__(self):
        """Initialize the analyzer with required NLTK components."""
        self.sia = SentimentIntensityAnalyzer()
        try:
            self.stop_words = set(stopwords.words('english'))
        except LookupError:
            nltk.download('stopwords')
            self.stop_words = set(stopwords.words('english'))
    
    def analyze_sentiment(self, text: str) -> Dict[str, Any]:
        """
        Analyze sentiment of text using multiple approaches.
        
        Args:
            text: Text to analyze
            
        Returns:
            Dictionary with sentiment scores and classification
        """
        # VADER sentiment analysis
        vader_scores = self.sia.polarity_scores(text)
        
        # TextBlob sentiment analysis
        blob = TextBlob(text)
        textblob_polarity = blob.sentiment.polarity
        textblob_subjectivity = blob.sentiment.subjectivity
        
        # Determine overall sentiment
        compound_score = vader_scores['compound']
        if compound_score >= 0.05:
            overall_sentiment = "positive"
        elif compound_score <= -0.05:
            overall_sentiment = "negative"
        else:
            overall_sentiment = "neutral"
        
        # Confidence based on compound score magnitude
        confidence = abs(compound_score)
        
        return {
            "overall_sentiment": overall_sentiment,
            "confidence": round(confidence, 3),
            "vader_scores": {
                "positive": round(vader_scores['pos'], 3),
                "negative": round(vader_scores['neg'], 3),
                "neutral": round(vader_scores['neu'], 3),
                "compound": round(vader_scores['compound'], 3)
            },
            "textblob_scores": {
                "polarity": round(textblob_polarity, 3),
                "subjectivity": round(textblob_subjectivity, 3)
            }
        }
    
    def extract_keywords(self, text: str, limit: int = 10) -> List[Dict[str, Any]]:
        """
        Extract keywords using TF-IDF and frequency analysis.
        
        Args:
            text: Text to analyze
            limit: Maximum number of keywords to return
            
        Returns:
            List of keywords with scores
        """
        # Clean and tokenize text
        cleaned_text = self._clean_text(text)
        words = word_tokenize(cleaned_text.lower())
        
        # Filter out stopwords and short words
        filtered_words = [
            word for word in words 
            if word not in self.stop_words 
            and len(word) > 2 
            and word.isalpha()
        ]
        
        # Frequency-based keywords
        word_freq = Counter(filtered_words)
        
        # TF-IDF based keywords (using the document as corpus)
        sentences = sent_tokenize(text)
        if len(sentences) > 1:
            try:
                vectorizer = TfidfVectorizer(
                    max_features=limit * 2,
                    stop_words='english',
                    ngram_range=(1, 2)
                )
                tfidf_matrix = vectorizer.fit_transform(sentences)
                feature_names = vectorizer.get_feature_names_out()
                tfidf_scores = tfidf_matrix.sum(axis=0).A1
                
                # Combine frequency and TF-IDF scores
                tfidf_keywords = dict(zip(feature_names, tfidf_scores))
            except:
                tfidf_keywords = {}
        else:
            tfidf_keywords = {}
        
        # Combine and rank keywords
        all_keywords = {}
        
        # Add frequency-based keywords
        for word, freq in word_freq.most_common(limit * 2):
            all_keywords[word] = {
                'frequency': freq,
                'tfidf': tfidf_keywords.get(word, 0),
                'combined_score': freq * 0.7 + tfidf_keywords.get(word, 0) * 0.3
            }
        
        # Add TF-IDF keywords that might not be in top frequency
        for word, score in tfidf_keywords.items():
            if word not in all_keywords:
                freq = filtered_words.count(word)
                all_keywords[word] = {
                    'frequency': freq,
                    'tfidf': score,
                    'combined_score': freq * 0.7 + score * 0.3
                }
        
        # Sort by combined score and return top keywords
        sorted_keywords = sorted(
            all_keywords.items(),
            key=lambda x: x[1]['combined_score'],
            reverse=True
        )
        
        result = []
        for word, scores in sorted_keywords[:limit]:
            result.append({
                'keyword': word,
                'frequency': scores['frequency'],
                'tfidf_score': round(scores['tfidf'], 4),
                'relevance_score': round(scores['combined_score'], 4)
            })
        
        return result
    
    def calculate_readability(self, text: str) -> Dict[str, Any]:
        """
        Calculate various readability metrics.
        
        Args:
            text: Text to analyze
            
        Returns:
            Dictionary with readability scores and interpretation
        """
        try:
            # Basic readability scores
            flesch_ease = flesch_reading_ease(text)
            flesch_grade = flesch_kincaid_grade(text)
            ari_score = automated_readability_index(text)
            
            # Interpret Flesch Reading Ease
            if flesch_ease >= 90:
                ease_level = "Very Easy"
            elif flesch_ease >= 80:
                ease_level = "Easy"
            elif flesch_ease >= 70:
                ease_level = "Fairly Easy"
            elif flesch_ease >= 60:
                ease_level = "Standard"
            elif flesch_ease >= 50:
                ease_level = "Fairly Difficult"
            elif flesch_ease >= 30:
                ease_level = "Difficult"
            else:
                ease_level = "Very Difficult"
            
            return {
                "flesch_reading_ease": round(flesch_ease, 2),
                "flesch_kincaid_grade": round(flesch_grade, 2),
                "automated_readability_index": round(ari_score, 2),
                "reading_level": ease_level,
                "grade_level": f"Grade {int(flesch_grade)}" if flesch_grade > 0 else "Elementary"
            }
            
        except Exception as e:
            return {
                "error": f"Could not calculate readability: {str(e)}",
                "flesch_reading_ease": 0,
                "flesch_kincaid_grade": 0,
                "automated_readability_index": 0,
                "reading_level": "Unknown",
                "grade_level": "Unknown"
            }
    
    def get_basic_stats(self, text: str) -> Dict[str, int]:
        """
        Calculate basic text statistics.
        
        Args:
            text: Text to analyze
            
        Returns:
            Dictionary with word count, sentence count, etc.
        """
        # Clean text for accurate counting
        cleaned_text = self._clean_text(text)
        
        # Count words
        words = word_tokenize(cleaned_text)
        word_count = len([word for word in words if word.isalpha()])
        
        # Count sentences
        sentences = sent_tokenize(text)
        sentence_count = len(sentences)
        
        # Count paragraphs (simple approach)
        paragraphs = [p.strip() for p in text.split('\n\n') if p.strip()]
        paragraph_count = len(paragraphs)
        
        # Count characters
        char_count = len(text)
        char_count_no_spaces = len(text.replace(' ', ''))
        
        # Average words per sentence
        avg_words_per_sentence = word_count / sentence_count if sentence_count > 0 else 0
        
        return {
            "word_count": word_count,
            "sentence_count": sentence_count,
            "paragraph_count": paragraph_count,
            "character_count": char_count,
            "character_count_no_spaces": char_count_no_spaces,
            "average_words_per_sentence": round(avg_words_per_sentence, 2)
        }
    
    def analyze_document(self, document: Dict[str, Any]) -> Dict[str, Any]:
        """
        Perform comprehensive analysis of a document.
        
        Args:
            document: Document dictionary with content and metadata
            
        Returns:
            Complete analysis results
        """
        content = document.get('content', '')
        
        if not content:
            return {"error": "No content to analyze"}
        
        # Perform all analyses
        sentiment = self.analyze_sentiment(content)
        keywords = self.extract_keywords(content)
        readability = self.calculate_readability(content)
        basic_stats = self.get_basic_stats(content)
        
        # Compile results
        analysis = {
            "document_id": document.get('id'),
            "title": document.get('title'),
            "analyzed_at": datetime.now().isoformat(),
            "sentiment_analysis": sentiment,
            "keywords": keywords,
            "readability": readability,
            "basic_statistics": basic_stats,
            "metadata": {
                "author": document.get('author'),
                "category": document.get('category'),
                "created_at": document.get('created_at', '').isoformat() if hasattr(document.get('created_at', ''), 'isoformat') else str(document.get('created_at', '')),
                "tags": document.get('tags', []),
                "language": document.get('language', 'en')
            }
        }
        
        return analysis
    
    def _clean_text(self, text: str) -> str:
        """
        Clean text for analysis by removing extra whitespace and formatting.
        
        Args:
            text: Raw text
            
        Returns:
            Cleaned text
        """
        # Remove extra whitespace
        text = re.sub(r'\s+', ' ', text)
        
        # Remove leading/trailing whitespace
        text = text.strip()
        
        return text 