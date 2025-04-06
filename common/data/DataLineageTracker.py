# DataLineageTracker.py
"""
Data Lineage Tracker
--------------------
Tracks the complete data lineage across IAROS using a directed graph.
Ensures full provenance and traceability for regulatory and audit purposes.
"""

import networkx as nx

class LineageTracker:
    def __init__(self):
        self.graph = nx.DiGraph()

    def track(self, data):
        node_id = data.get('id')
        if not node_id:
            raise ValueError("Data must include an 'id' field")
        self.graph.add_node(node_id, metadata={
            'origin': data.get('source'),
            'processing_steps': data.get('processing_steps', [])
        })
        return node_id

    def add_transformation(self, from_id, to_id, transformation):
        self.graph.add_edge(from_id, to_id, transformation=transformation)

    def get_lineage(self, node_id):
        return nx.descendants(self.graph, node_id)
