#!/usr/bin/env python3
"""
Document ingestion script for Filo's RAG system.
This script processes business documents and adds them to the Qdrant knowledge base.
"""

import os
import sys
import argparse
import hashlib
import logging
from pathlib import Path
from typing import List, Dict, Any
import requests
import json
from urllib.parse import urljoin

# Configure logging
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

class DocumentIngestor:
    """Handles document ingestion into Qdrant knowledge base."""
    
    def __init__(self, qdrant_url: str, api_key: str, collection_name: str):
        self.qdrant_url = qdrant_url
        self.api_key = api_key
        self.collection_name = collection_name
        self.headers = {
            "Content-Type": "application/json",
            "api-key": api_key
        }
        
    def create_collection(self) -> bool:
        """Create the knowledge base collection in Qdrant if it doesn't exist."""
        url = urljoin(self.qdrant_url, f"/collections/{self.collection_name}")
        
        # Define collection configuration
        config = {
            "vectors": {
                "size": 1536,  # Default embedding size (adjust as needed)
                "distance": "Cosine"
            }
        }
        
        try:
            response = requests.put(url, headers=self.headers, json=config)
            if response.status_code == 200:
                logger.info(f"Collection '{self.collection_name}' created successfully")
                return True
            elif response.status_code == 409:
                # Collection already exists
                logger.info(f"Collection '{self.collection_name}' already exists")
                return True
            else:
                logger.error(f"Failed to create collection: {response.status_code} - {response.text}")
                return False
        except Exception as e:
            logger.error(f"Error creating collection: {str(e)}")
            return False
    
    def calculate_embedding(self, text: str) -> List[float]:
        """
        Calculate a simple embedding for the given text.
        Note: This is a placeholder implementation. In production, use a proper embedding model.
        """
        # This is a simple hash-based embedding for demonstration purposes
        # In a real implementation, you would use a model like Sentence Transformers
        import numpy as np
        
        # Convert text to a numeric representation
        text_bytes = text.encode('utf-8')
        hash_obj = hashlib.sha256(text_bytes)
        hash_hex = hash_obj.hexdigest()
        
        # Convert hex to floats
        embedding = []
        for i in range(0, len(hash_hex), 8):
            chunk = hash_hex[i:i+8]
            if len(chunk) == 8:  # Ensure we have a full chunk
                # Convert hex chunk to float
                val = int(chunk, 16) / (2**32)  # Normalize to 0-1 range
                embedding.append(val)
        
        # Pad or truncate to desired size (1536 dimensions)
        while len(embedding) < 1536:
            embedding.append(0.0)
        embedding = embedding[:1536]
        
        return embedding
    
    def add_document(self, doc_id: str, content: str, title: str = "", source: str = "") -> bool:
        """Add a single document to the Qdrant collection."""
        url = urljoin(self.qdrant_url, "/points")
        
        # Calculate embedding for the document content
        vector = self.calculate_embedding(content)
        
        # Prepare the point data
        point = {
            "id": doc_id,
            "vector": vector,
            "payload": {
                "content": content,
                "title": title,
                "source": source
            }
        }
        
        payload = {
            "collection_name": self.collection_name,
            "points": [point],
            "wait": True  # Wait for operation to complete
        }
        
        try:
            response = requests.put(url, headers=self.headers, json=payload)
            if response.status_code == 200:
                logger.info(f"Document '{doc_id}' added successfully")
                return True
            else:
                logger.error(f"Failed to add document: {response.status_code} - {response.text}")
                return False
        except Exception as e:
            logger.error(f"Error adding document: {str(e)}")
            return False
    
    def process_text_file(self, file_path: str) -> List[Dict[str, Any]]:
        """Process a text file and split it into chunks."""
        with open(file_path, 'r', encoding='utf-8') as f:
            content = f.read()
        
        # Simple chunking strategy - split by paragraphs
        paragraphs = content.split('\n\n')
        
        chunks = []
        for i, paragraph in enumerate(paragraphs):
            if paragraph.strip():  # Skip empty paragraphs
                chunk_id = f"{Path(file_path).stem}_chunk_{i}"
                chunks.append({
                    "id": chunk_id,
                    "content": paragraph.strip(),
                    "title": f"{Path(file_path).stem} - Chunk {i}",
                    "source": str(file_path)
                })
        
        return chunks
    
    def process_directory(self, directory_path: str) -> List[Dict[str, Any]]:
        """Process all text files in a directory."""
        directory = Path(directory_path)
        all_chunks = []
        
        for txt_file in directory.glob("**/*.txt"):
            logger.info(f"Processing file: {txt_file}")
            chunks = self.process_text_file(txt_file)
            all_chunks.extend(chunks)
        
        return all_chunks
    
    def ingest_documents(self, source_path: str) -> bool:
        """Ingest documents from a file or directory into Qdrant."""
        # Create collection if it doesn't exist
        if not self.create_collection():
            return False
        
        # Process source (file or directory)
        source = Path(source_path)
        if source.is_file() and source.suffix.lower() == '.txt':
            chunks = self.process_text_file(source_path)
        elif source.is_dir():
            chunks = self.process_directory(source_path)
        else:
            logger.error(f"Source must be a .txt file or directory: {source_path}")
            return False
        
        logger.info(f"Found {len(chunks)} document chunks to ingest")
        
        # Add each chunk to Qdrant
        success_count = 0
        for chunk in chunks:
            if self.add_document(
                doc_id=chunk["id"],
                content=chunk["content"],
                title=chunk["title"],
                source=chunk["source"]
            ):
                success_count += 1
        
        logger.info(f"Successfully ingested {success_count}/{len(chunks)} documents")
        return success_count == len(chunks)

def main():
    parser = argparse.ArgumentParser(description="Ingest business documents into Filo's knowledge base")
    parser.add_argument("--qdrant-url", default=os.getenv("QDRANT_URL", "http://localhost:6333"),
                        help="Qdrant server URL (default: from QDRANT_URL env var or http://localhost:6333)")
    parser.add_argument("--api-key", default=os.getenv("QDRANT_API_KEY", ""),
                        help="Qdrant API key (default: from QDRANT_API_KEY env var)")
    parser.add_argument("--collection", default=os.getenv("QDRANT_COLLECTION_NAME", "knowledge_base"),
                        help="Qdrant collection name (default: from QDRANT_COLLECTION_NAME env var or 'knowledge_base')")
    parser.add_argument("source", help="Path to text file or directory containing documents to ingest")
    
    args = parser.parse_args()
    
    if not args.api_key:
        logger.error("QDRANT_API_KEY must be provided either as argument or environment variable")
        sys.exit(1)
    
    # Create document ingestor
    ingestor = DocumentIngestor(args.qdrant_url, args.api_key, args.collection)
    
    # Perform ingestion
    success = ingestor.ingest_documents(args.source)
    
    if success:
        logger.info("Document ingestion completed successfully!")
        sys.exit(0)
    else:
        logger.error("Document ingestion failed!")
        sys.exit(1)

if __name__ == "__main__":
    main()