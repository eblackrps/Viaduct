import { useEffect, useMemo, useRef, useState } from "react";
import * as d3 from "d3";
import { Search } from "lucide-react";
import { getGraph } from "../api";
import { StatusBadge } from "./primitives/StatusBadge";
import type { DependencyGraph as DependencyGraphModel, GraphEdge, GraphNode } from "../types";

type GraphSimulationNode = GraphNode & d3.SimulationNodeDatum;
type GraphSimulationLink = GraphEdge & d3.SimulationLinkDatum<GraphSimulationNode>;

interface GraphFilterState {
  nodeTypes: Record<GraphNode["type"], boolean>;
  platform: string;
  search: string;
}

export function DependencyGraph() {
  const svgRef = useRef<SVGSVGElement | null>(null);
  const [graph, setGraph] = useState<DependencyGraphModel | null>(null);
  const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null);
  const [filters, setFilters] = useState<GraphFilterState>({
    nodeTypes: {
      vm: true,
      network: true,
      datastore: true,
      "backup-job": true,
    },
    platform: "all",
    search: "",
  });
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    setLoading(true);
    setSelectedNodeId(null);
    getGraph()
      .then((result) => {
        setGraph(result);
        setError(null);
      })
      .catch((reason: Error) => setError(reason.message))
      .finally(() => setLoading(false));
  }, []);

  const visibleGraph = useMemo(() => {
    const baseNodes = (graph?.nodes ?? []).filter((node) => {
      if (!filters.nodeTypes[node.type]) {
        return false;
      }
      if (filters.platform !== "all" && node.platform && node.platform !== filters.platform) {
        return false;
      }
      return true;
    });

    if (filters.search.trim() === "") {
      const nodeIDs = new Set(baseNodes.map((node) => node.id));
      return {
        nodes: baseNodes,
        edges: (graph?.edges ?? []).filter((edge) => nodeIDs.has(edge.source) && nodeIDs.has(edge.target)),
      };
    }

    const baseNodeIDs = new Set(baseNodes.map((node) => node.id));
    const matchedNodeIDs = new Set(
      baseNodes
        .filter((node) => matchesSearch(node, filters.search))
        .map((node) => node.id),
    );
    const contextualIDs = new Set<string>(matchedNodeIDs);

    for (const edge of graph?.edges ?? []) {
      if (matchedNodeIDs.has(edge.source) || matchedNodeIDs.has(edge.target)) {
        if (baseNodeIDs.has(edge.source)) {
          contextualIDs.add(edge.source);
        }
        if (baseNodeIDs.has(edge.target)) {
          contextualIDs.add(edge.target);
        }
      }
    }

    const nodes = baseNodes.filter((node) => contextualIDs.has(node.id));
    const nodeIDs = new Set(nodes.map((node) => node.id));
    const edges = (graph?.edges ?? []).filter((edge) => nodeIDs.has(edge.source) && nodeIDs.has(edge.target));
    return { nodes, edges };
  }, [filters, graph]);

  const adjacency = useMemo(() => buildAdjacency(visibleGraph), [visibleGraph]);
  const platformOptions = useMemo(() => {
    const options = new Set<string>();
    for (const node of graph?.nodes ?? []) {
      if (node.platform) {
        options.add(node.platform);
      }
    }
    return Array.from(options).sort();
  }, [graph]);
  const selectedNode =
    visibleGraph.nodes.find((node) => node.id === selectedNodeId) ??
    graph?.nodes.find((node) => node.id === selectedNodeId) ??
    null;
  const relatedNodes = useMemo(() => {
    if (!selectedNode) {
      return [];
    }
    return adjacency.get(selectedNode.id) ?? [];
  }, [adjacency, selectedNode]);
  const topWorkloads = useMemo(
    () =>
      visibleGraph.nodes
        .filter((node) => node.type === "vm")
        .map((node) => ({ node, degree: (adjacency.get(node.id) ?? []).length }))
        .sort((left, right) => right.degree - left.degree || left.node.label.localeCompare(right.node.label))
        .slice(0, 8),
    [adjacency, visibleGraph.nodes],
  );

  useEffect(() => {
    if (selectedNodeId && !visibleGraph.nodes.some((node) => node.id === selectedNodeId)) {
      setSelectedNodeId(null);
    }
  }, [selectedNodeId, visibleGraph.nodes]);

  useEffect(() => {
    if (!svgRef.current) {
      return;
    }

    const width = 960;
    const height = 560;
    const svg = d3.select(svgRef.current);
    svg.selectAll("*").remove();

    if (visibleGraph.nodes.length === 0) {
      return;
    }

    const root = svg.attr("viewBox", `0 0 ${width} ${height}`);
    const canvas = root.append("g");
    const simulationNodes: GraphSimulationNode[] = visibleGraph.nodes.map((node) => ({ ...node }));
    const simulationLinks: GraphSimulationLink[] = visibleGraph.edges.map((edge) => ({ ...edge }));

    root.call(
      d3.zoom<SVGSVGElement, unknown>().scaleExtent([0.5, 2]).on("zoom", (event: d3.D3ZoomEvent<SVGSVGElement, unknown>) => {
        canvas.attr("transform", event.transform.toString());
      }),
    );

    const simulation = d3
      .forceSimulation<GraphSimulationNode>(simulationNodes)
      .force(
        "link",
        d3
          .forceLink<GraphSimulationNode, GraphSimulationLink>(simulationLinks)
          .id((node) => node.id)
          .distance((edge) => (edge.type === "backup" ? 150 : 120)),
      )
      .force("charge", d3.forceManyBody().strength(-280))
      .force("center", d3.forceCenter(width / 2, height / 2));

    const edges = canvas
      .append("g")
      .selectAll("line")
      .data(simulationLinks)
      .join("line")
      .attr("stroke", (edge: GraphSimulationLink) => edgeColor(edge))
      .attr("stroke-width", 1.75)
      .attr("stroke-opacity", 0.65);

    const nodes = canvas
      .append("g")
      .selectAll<SVGCircleElement, GraphSimulationNode>("circle")
      .data(simulationNodes)
      .join("circle")
      .attr("r", (node) => (node.id === selectedNodeId ? 18 : node.type === "vm" ? 14 : 12))
      .attr("fill", (node: GraphSimulationNode) => nodeColor(node))
      .attr("stroke", (node) => (node.id === selectedNodeId ? "#0f172a" : "#ffffff"))
      .attr("stroke-width", (node) => (node.id === selectedNodeId ? 3 : 2))
      .style("cursor", "pointer")
      .call(
        d3
          .drag<SVGCircleElement, GraphSimulationNode>()
          .on("start", (event: d3.D3DragEvent<SVGCircleElement, GraphSimulationNode, GraphSimulationNode>, node) => {
            if (!event.active) {
              simulation.alphaTarget(0.3).restart();
            }
            node.fx = node.x;
            node.fy = node.y;
          })
          .on("drag", (event: d3.D3DragEvent<SVGCircleElement, GraphSimulationNode, GraphSimulationNode>, node) => {
            node.fx = event.x;
            node.fy = event.y;
          })
          .on("end", (event: d3.D3DragEvent<SVGCircleElement, GraphSimulationNode, GraphSimulationNode>, node) => {
            if (!event.active) {
              simulation.alphaTarget(0);
            }
            node.fx = null;
            node.fy = null;
          }),
      )
      .on("click", (_event: MouseEvent, node: GraphSimulationNode) => setSelectedNodeId(node.id));

    nodes.append("title").text((node: GraphSimulationNode) => node.label);

    const labels = canvas
      .append("g")
      .selectAll("text")
      .data(simulationNodes)
      .join("text")
      .attr("font-size", 12)
      .attr("fill", "#0f172a")
      .text((node: GraphSimulationNode) => node.label);

    simulation.on("tick", () => {
      edges
        .attr("x1", (edge) => (typeof edge.source === "object" ? edge.source.x ?? 0 : 0))
        .attr("y1", (edge) => (typeof edge.source === "object" ? edge.source.y ?? 0 : 0))
        .attr("x2", (edge) => (typeof edge.target === "object" ? edge.target.x ?? 0 : 0))
        .attr("y2", (edge) => (typeof edge.target === "object" ? edge.target.y ?? 0 : 0));

      nodes.attr("cx", (node) => node.x ?? 0).attr("cy", (node) => node.y ?? 0);
      labels.attr("x", (node) => (node.x ?? 0) + 18).attr("y", (node) => (node.y ?? 0) + 4);
    });

    return () => {
      simulation.stop();
    };
  }, [selectedNodeId, visibleGraph]);

  return (
    <section className="grid gap-5 xl:grid-cols-[1.65fr_0.95fr]">
      <div className="panel p-5">
        <div className="flex flex-col gap-4">
          <div className="flex flex-wrap items-center gap-2">
            <StatusBadge tone="info">{visibleGraph.nodes.length} nodes</StatusBadge>
            <StatusBadge tone="neutral">
              {visibleGraph.nodes.filter((node) => node.type === "vm").length} workloads
            </StatusBadge>
            <StatusBadge tone="neutral">
              {visibleGraph.edges.filter((edge) => edge.type === "storage").length} storage links
            </StatusBadge>
            <StatusBadge tone="neutral">
              {visibleGraph.edges.filter((edge) => edge.type === "backup").length} backup links
            </StatusBadge>
          </div>

          <div className="grid gap-3 xl:grid-cols-[minmax(0,1fr)_auto_auto]">
            <label className="flex items-center gap-2 rounded-full border border-slate-200 bg-white px-4 py-2 text-sm text-slate-500">
              <Search className="h-4 w-4" />
              <input
                className="w-full border-none bg-transparent outline-none"
                placeholder="Search workloads, networks, datastores, or backup jobs"
                value={filters.search}
                onChange={(event) => setFilters((current) => ({ ...current, search: event.target.value }))}
              />
            </label>
            <select
              className="rounded-full border border-slate-200 px-4 py-2 text-sm text-slate-700"
              value={filters.platform}
              onChange={(event) => setFilters((current) => ({ ...current, platform: event.target.value }))}
            >
              <option value="all">All platforms</option>
              {platformOptions.map((platform) => (
                <option key={platform} value={platform}>
                  {platform}
                </option>
              ))}
            </select>
            <div className="flex flex-wrap gap-2">
              {Object.entries(filters.nodeTypes).map(([key, enabled]) => (
                <label key={key} className="inline-flex items-center gap-2 rounded-full border border-slate-200 bg-white px-3 py-2 text-sm text-slate-600">
                  <input
                    type="checkbox"
                    checked={enabled}
                    onChange={(event) =>
                      setFilters((current) => ({
                        ...current,
                        nodeTypes: { ...current.nodeTypes, [key as GraphNode["type"]]: event.target.checked },
                      }))
                    }
                  />
                  {key}
                </label>
              ))}
            </div>
          </div>
        </div>

        {error && <p className="mt-4 rounded-2xl bg-rose-50 px-4 py-3 text-sm text-rose-700">{error}</p>}

        {loading ? (
          <div className="mt-5 rounded-2xl border border-dashed border-slate-300 px-4 py-6 text-sm text-slate-500">
            Loading dependency graph...
          </div>
        ) : visibleGraph.nodes.length === 0 ? (
          <div className="mt-5 rounded-2xl border border-dashed border-slate-300 px-4 py-6 text-sm text-slate-500">
            No dependency nodes match the current filters.
          </div>
        ) : (
          <div className="mt-5 grid gap-5 xl:grid-cols-[minmax(0,1fr)_260px]">
            <svg ref={svgRef} className="h-[560px] w-full rounded-3xl bg-slate-50" />
            <div className="space-y-3 rounded-3xl bg-slate-50 p-4">
              <p className="font-semibold text-ink">Most connected workloads</p>
              <p className="text-sm text-slate-500">
                Use the graph plus this list to focus on workloads with the most adjacent assets.
              </p>
              <div className="space-y-2">
                {topWorkloads.map(({ node, degree }) => (
                  <button
                    key={node.id}
                    type="button"
                    onClick={() => setSelectedNodeId(node.id)}
                    className={`w-full rounded-2xl px-3 py-3 text-left text-sm transition ${
                      selectedNodeId === node.id ? "bg-white text-ink shadow-sm" : "bg-white/70 text-slate-700 hover:bg-white"
                    }`}
                  >
                    <p className="font-semibold">{node.label}</p>
                    <p className="mt-1 text-xs text-slate-500">{degree} direct relationship(s)</p>
                  </button>
                ))}
                {topWorkloads.length === 0 && <p className="text-sm text-slate-500">No workloads are visible in the current graph scope.</p>}
              </div>
            </div>
          </div>
        )}
      </div>

      <aside className="panel p-5">
        <p className="font-display text-2xl text-ink">Dependency detail</p>
        {loading && <p className="mt-3 text-sm text-slate-500">Loading dependency metadata...</p>}
        {!loading && visibleGraph.nodes.length === 0 && <p className="mt-3 text-sm text-slate-500">No nodes are available to inspect.</p>}
        {!loading && visibleGraph.nodes.length > 0 && !selectedNode && (
          <p className="mt-3 text-sm text-slate-500">
            Select a node in the graph or the workload index to inspect direct relationships and metadata.
          </p>
        )}
        {selectedNode && (
          <div className="mt-4 space-y-4 text-sm text-slate-600">
            <div className="rounded-3xl bg-slate-50 px-4 py-4">
              <div className="flex flex-wrap items-center gap-2">
                <p className="font-semibold text-ink">{selectedNode.label}</p>
                <StatusBadge tone={selectedNode.type === "vm" ? "info" : "neutral"}>{selectedNode.type}</StatusBadge>
                {selectedNode.platform && <StatusBadge tone="neutral">{selectedNode.platform}</StatusBadge>}
              </div>
              <p className="mt-2 text-slate-500">
                {relatedNodes.length} direct relationship(s) in the current filtered graph scope.
              </p>
            </div>

            {selectedNode.metadata && Object.keys(selectedNode.metadata).length > 0 && (
              <div className="space-y-2">
                {Object.entries(selectedNode.metadata).map(([key, value]) => (
                  <div key={key} className="rounded-2xl bg-slate-50 px-4 py-3">
                    <p className="text-xs uppercase tracking-[0.18em] text-slate-500">{key}</p>
                    <p className="mt-1 font-semibold text-ink">{value}</p>
                  </div>
                ))}
              </div>
            )}

            <div>
              <p className="text-xs uppercase tracking-[0.18em] text-slate-500">Direct relationships</p>
              <div className="mt-2 space-y-2">
                {relatedNodes.length > 0 ? (
                  relatedNodes.map((relation) => (
                    <button
                      key={`${selectedNode.id}:${relation.edge.label}:${relation.node.id}`}
                      type="button"
                      onClick={() => setSelectedNodeId(relation.node.id)}
                      className="w-full rounded-2xl bg-slate-50 px-4 py-3 text-left transition hover:bg-slate-100"
                    >
                      <div className="flex items-center justify-between gap-3">
                        <p className="font-semibold text-ink">{relation.node.label}</p>
                        <StatusBadge tone={edgeTone(relation.edge.type)}>{relation.edge.type}</StatusBadge>
                      </div>
                      <p className="mt-2 text-slate-500">{relation.edge.label}</p>
                    </button>
                  ))
                ) : (
                  <p className="rounded-2xl bg-slate-50 px-4 py-3 text-sm text-slate-500">
                    No direct relationships are visible for this node with the current filters.
                  </p>
                )}
              </div>
            </div>
          </div>
        )}
      </aside>
    </section>
  );
}

function buildAdjacency(graph: { nodes: GraphNode[]; edges: GraphEdge[] }): Map<string, Array<{ edge: GraphEdge; node: GraphNode }>> {
  const nodeById = new Map(graph.nodes.map((node) => [node.id, node]));
  const adjacency = new Map<string, Array<{ edge: GraphEdge; node: GraphNode }>>();

  for (const edge of graph.edges) {
    const sourceNode = nodeById.get(edge.source);
    const targetNode = nodeById.get(edge.target);
    if (!sourceNode || !targetNode) {
      continue;
    }

    adjacency.set(edge.source, [...(adjacency.get(edge.source) ?? []), { edge, node: targetNode }]);
    adjacency.set(edge.target, [...(adjacency.get(edge.target) ?? []), { edge, node: sourceNode }]);
  }

  return adjacency;
}

function matchesSearch(node: GraphNode, search: string): boolean {
  const text = search.trim().toLowerCase();
  if (!text) {
    return true;
  }

  return [node.label, node.type, node.platform ?? "", ...Object.entries(node.metadata ?? {}).flat()]
    .join(" ")
    .toLowerCase()
    .includes(text);
}

function nodeColor(node: GraphNode): string {
  switch (node.type) {
    case "vm":
      return "#1f4e79";
    case "network":
      return "#2e6f40";
    case "datastore":
      return "#e56b1f";
    case "backup-job":
      return "#475569";
    default:
      return "#64748b";
  }
}

function edgeColor(edge: GraphEdge): string {
  switch (edge.type) {
    case "network":
      return "#2e6f40";
    case "storage":
      return "#e56b1f";
    case "backup":
      return "#475569";
    default:
      return "#94a3b8";
  }
}

function edgeTone(edgeType: GraphEdge["type"]): "neutral" | "info" | "warning" {
  switch (edgeType) {
    case "network":
      return "info";
    case "storage":
      return "warning";
    default:
      return "neutral";
  }
}
