import { useEffect, useMemo, useRef, useState } from "react";
import * as d3 from "d3";
import { getGraph } from "../api";
import type { DependencyGraph as DependencyGraphModel, GraphEdge, GraphFilters, GraphNode } from "../types";

export function DependencyGraph() {
  const svgRef = useRef<SVGSVGElement | null>(null);
  const [graph, setGraph] = useState<DependencyGraphModel | null>(null);
  const [selectedNode, setSelectedNode] = useState<GraphNode | null>(null);
  const [filters, setFilters] = useState<GraphFilters>({
    nodeTypes: {
      vm: true,
      network: true,
      datastore: true,
      "backup-job": true,
    },
    platform: "all",
  });
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    getGraph()
      .then((result) => {
        setGraph(result);
        setError(null);
      })
      .catch((err: Error) => setError(err.message));
  }, []);

  const visibleGraph = useMemo(() => {
    const nodes = (graph?.nodes ?? []).filter((node) => {
      if (!filters.nodeTypes[node.type]) {
        return false;
      }
      if (filters.platform !== "all" && node.platform && node.platform !== filters.platform) {
        return false;
      }
      return true;
    });
    const nodeIDs = new Set(nodes.map((node) => node.id));
    const edges = (graph?.edges ?? []).filter((edge) => nodeIDs.has(edge.source) && nodeIDs.has(edge.target));
    return { nodes, edges };
  }, [filters, graph]);

  useEffect(() => {
    if (!svgRef.current) {
      return;
    }

    const width = 900;
    const height = 520;
    const svg = d3.select(svgRef.current);
    svg.selectAll("*").remove();

    const root = svg.attr("viewBox", `0 0 ${width} ${height}`);
    const canvas = root.append("g");

    root.call(
      d3.zoom<SVGSVGElement, unknown>().scaleExtent([0.5, 2]).on("zoom", (event: d3.D3ZoomEvent<SVGSVGElement, unknown>) => {
        canvas.attr("transform", event.transform.toString());
      }),
    );

    const simulation = d3
      .forceSimulation(visibleGraph.nodes as d3.SimulationNodeDatum[])
      .force("link", d3.forceLink(visibleGraph.edges as d3.SimulationLinkDatum<d3.SimulationNodeDatum>[]).id((d: any) => d.id).distance(120))
      .force("charge", d3.forceManyBody().strength(-240))
      .force("center", d3.forceCenter(width / 2, height / 2));

    const edges = canvas
      .append("g")
      .selectAll("line")
      .data(visibleGraph.edges)
      .join("line")
      .attr("stroke", (edge: GraphEdge) => edgeColor(edge))
      .attr("stroke-width", 1.5)
      .attr("stroke-opacity", 0.65);

    const nodes = canvas
      .append("g")
      .selectAll<SVGCircleElement, GraphNode>("circle")
      .data(visibleGraph.nodes)
      .join("circle")
      .attr("r", 14)
      .attr("fill", (node: GraphNode) => nodeColor(node))
      .attr("stroke", "#ffffff")
      .attr("stroke-width", 2)
      .style("cursor", "pointer")
      .call(
        d3
          .drag<SVGCircleElement, GraphNode>()
          .on("start", (event: d3.D3DragEvent<SVGCircleElement, GraphNode, GraphNode>, node: any) => {
            if (!event.active) {
              simulation.alphaTarget(0.3).restart();
            }
            node.fx = node.x;
            node.fy = node.y;
          })
          .on("drag", (event: d3.D3DragEvent<SVGCircleElement, GraphNode, GraphNode>, node: any) => {
            node.fx = event.x;
            node.fy = event.y;
          })
          .on("end", (event: d3.D3DragEvent<SVGCircleElement, GraphNode, GraphNode>, node: any) => {
            if (!event.active) {
              simulation.alphaTarget(0);
            }
            node.fx = null;
            node.fy = null;
          }),
      )
      .on("click", (_event: MouseEvent, node: GraphNode) => setSelectedNode(node));

    nodes.append("title").text((node: GraphNode) => node.label);

    const labels = canvas
      .append("g")
      .selectAll("text")
      .data(visibleGraph.nodes)
      .join("text")
      .attr("font-size", 12)
      .attr("fill", "#0f172a")
      .text((node: GraphNode) => node.label);

    simulation.on("tick", () => {
      edges
        .attr("x1", (edge: any) => edge.source.x)
        .attr("y1", (edge: any) => edge.source.y)
        .attr("x2", (edge: any) => edge.target.x)
        .attr("y2", (edge: any) => edge.target.y);

      nodes.attr("cx", (node: any) => node.x).attr("cy", (node: any) => node.y);
      labels.attr("x", (node: any) => node.x + 18).attr("y", (node: any) => node.y + 4);
    });

    return () => {
      simulation.stop();
    };
  }, [visibleGraph]);

  return (
    <section className="grid gap-5 xl:grid-cols-[1.6fr_0.8fr]">
      <div className="panel p-5">
        <div className="mb-4 flex flex-wrap items-center gap-4">
          {Object.entries(filters.nodeTypes).map(([key, enabled]) => (
            <label key={key} className="inline-flex items-center gap-2 text-sm text-slate-600">
              <input
                type="checkbox"
                checked={enabled}
                onChange={(event) =>
                  setFilters((current) => ({
                    ...current,
                    nodeTypes: { ...current.nodeTypes, [key]: event.target.checked },
                  }))
                }
              />
              {key}
            </label>
          ))}
          <select className="rounded-full border border-slate-200 px-4 py-2 text-sm text-slate-700" value={filters.platform} onChange={(event) => setFilters((current) => ({ ...current, platform: event.target.value }))}>
            <option value="all">All Platforms</option>
            <option value="vmware">VMware</option>
            <option value="proxmox">Proxmox</option>
            <option value="hyperv">Hyper-V</option>
          </select>
        </div>
        {error && <p className="mb-4 rounded-2xl bg-rose-50 px-4 py-3 text-sm text-rose-700">{error}</p>}
        <svg ref={svgRef} className="h-[520px] w-full rounded-3xl bg-slate-50" />
      </div>

      <aside className="panel p-5">
        <p className="font-display text-2xl text-ink">Node Detail</p>
        {!selectedNode && <p className="mt-3 text-sm text-slate-500">Click any node in the graph to inspect its metadata and attached relationships.</p>}
        {selectedNode && (
          <div className="mt-4 space-y-3 text-sm text-slate-600">
            <div>
              <p className="font-semibold text-ink">{selectedNode.label}</p>
              <p className="text-slate-500">{selectedNode.type}</p>
            </div>
            {selectedNode.metadata &&
              Object.entries(selectedNode.metadata).map(([key, value]) => (
                <div key={key} className="rounded-2xl bg-slate-50 px-4 py-3">
                  <p className="text-xs uppercase tracking-[0.18em] text-slate-500">{key}</p>
                  <p className="mt-1 font-semibold text-ink">{value}</p>
                </div>
              ))}
          </div>
        )}
      </aside>
    </section>
  );
}

function nodeColor(node: GraphNode) {
  switch (node.type) {
    case "vm":
      return "#1f4e79";
    case "network":
      return "#2e6f40";
    case "datastore":
      return "#e56b1f";
    case "backup-job":
      return "#7c3aed";
    default:
      return "#64748b";
  }
}

function edgeColor(edge: GraphEdge) {
  switch (edge.type) {
    case "network":
      return "#2e6f40";
    case "storage":
      return "#e56b1f";
    case "backup":
      return "#7c3aed";
    default:
      return "#94a3b8";
  }
}
