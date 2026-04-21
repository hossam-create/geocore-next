"use client";
import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { adsApi } from "@/lib/api";
import PageHeader from "@/components/shared/PageHeader";
import StatusBadge from "@/components/shared/StatusBadge";
import { Plus, Pencil, Trash2, X, Eye, MousePointerClick } from "lucide-react";

interface Ad { id:string; title:string; image_url:string; link_url:string; placement:string; position:number; enabled:boolean; start_date:string|null; end_date:string|null; click_count:number; view_count:number; created_at:string; }
const PL = [{value:"hero",label:"Hero Carousel"},{value:"sidebar",label:"Sidebar"},{value:"category",label:"Category Spotlight"},{value:"listing_footer",label:"Listing Footer"}];
const EMPTY:Partial<Ad> = {title:"",image_url:"",link_url:"",placement:"hero",position:0,enabled:true,start_date:null,end_date:null};
function status(a:Ad){if(!a.enabled)return"disabled";const n=new Date();if(a.start_date&&new Date(a.start_date)>n)return"scheduled";if(a.end_date&&new Date(a.end_date)<n)return"expired";return"active";}
function plLabel(p:string){return PL.find(x=>x.value===p)?.label??p;}

export default function BannersPage(){
  const qc=useQueryClient(),[editing,setEditing]=useState<Partial<Ad>|null>(null),[creating,setCreating]=useState(false),[fp,setFp]=useState(""),[fs,setFs]=useState("");
  const params:Record<string,string>={};if(fp)params.placement=fp;if(fs)params.status=fs;
  const{data:ads=[],isLoading}=useQuery({queryKey:["ads",params],queryFn:()=>adsApi.list(params)});
  const cMut=useMutation({mutationFn:(d:Record<string,unknown>)=>adsApi.create(d),onSuccess:()=>{qc.invalidateQueries({queryKey:["ads"]});setCreating(false);setEditing(null);}});
  const uMut=useMutation({mutationFn:({id,...d}:Record<string,unknown>)=>adsApi.update(id as string,d),onSuccess:()=>{qc.invalidateQueries({queryKey:["ads"]});setEditing(null);}});
  const dMut=useMutation({mutationFn:(id:string)=>adsApi.delete(id),onSuccess:()=>qc.invalidateQueries({queryKey:["ads"]})});
  const tMut=useMutation({mutationFn:(id:string)=>adsApi.toggle(id),onSuccess:()=>qc.invalidateQueries({queryKey:["ads"]})});
  const save=()=>{if(!editing)return;if(creating)cMut.mutate(editing as Record<string,unknown>);else if(editing.id)uMut.mutate({id:editing.id,...editing} as Record<string,unknown>);};
  const al=ads as Ad[],tot=al.length,act=al.filter(a=>status(a)==="active").length,sch=al.filter(a=>status(a)==="scheduled").length,exp=al.filter(a=>status(a)==="expired").length;

  return(<div>
    <PageHeader title="Banner Ads" description="Manage homepage banners and promotional placements" actions={<button onClick={()=>{setEditing({...EMPTY});setCreating(true);}}className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm font-medium text-white"style={{background:"var(--color-brand)"}}><Plus className="w-4 h-4"/>New Banner</button>}/>
    <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6">
      {[{l:"Total",v:tot,c:"text-slate-900"},{l:"Active",v:act,c:"text-emerald-600"},{l:"Scheduled",v:sch,c:"text-blue-600"},{l:"Expired",v:exp,c:"text-slate-400"}].map(s=>(<div key={s.l}className="p-4 rounded-xl border border-slate-200 bg-white"><p className="text-xs text-slate-500 mb-1">{s.l}</p><p className={`text-2xl font-bold ${s.c}`}>{s.v}</p></div>))}
    </div>
    <div className="flex gap-3 mb-4">
      <select value={fp}onChange={e=>setFp(e.target.value)}className="border border-slate-200 rounded-lg px-3 py-1.5 text-sm bg-white"><option value="">All Placements</option>{PL.map(p=><option key={p.value}value={p.value}>{p.label}</option>)}</select>
      <select value={fs}onChange={e=>setFs(e.target.value)}className="border border-slate-200 rounded-lg px-3 py-1.5 text-sm bg-white"><option value="">All Status</option><option value="active">Active</option><option value="scheduled">Scheduled</option><option value="expired">Expired</option><option value="disabled">Disabled</option></select>
    </div>
    {editing&&(<div className="p-4 mb-4 rounded-xl border border-slate-200 bg-white">
      <div className="flex items-center justify-between mb-3"><h3 className="font-semibold text-sm">{creating?"New Banner":"Edit Banner"}</h3><button onClick={()=>{setEditing(null);setCreating(false);}}><X className="w-4 h-4 text-slate-400"/></button></div>
      <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
        <input placeholder="Title"value={editing.title??""}onChange={e=>setEditing({...editing,title:e.target.value})}className="border rounded-lg px-3 py-2 text-sm"/>
        <select value={editing.placement??"hero"}onChange={e=>setEditing({...editing,placement:e.target.value})}className="border rounded-lg px-3 py-2 text-sm">{PL.map(p=><option key={p.value}value={p.value}>{p.label}</option>)}</select>
        <input placeholder="Image URL"value={editing.image_url??""}onChange={e=>setEditing({...editing,image_url:e.target.value})}className="border rounded-lg px-3 py-2 text-sm"/>
        <input placeholder="Link URL (optional)"value={editing.link_url??""}onChange={e=>setEditing({...editing,link_url:e.target.value})}className="border rounded-lg px-3 py-2 text-sm"/>
        <input type="number"placeholder="Position"value={editing.position??0}onChange={e=>setEditing({...editing,position:parseInt(e.target.value)||0})}className="border rounded-lg px-3 py-2 text-sm"/>
        <label className="flex items-center gap-2 text-sm"><input type="checkbox"checked={editing.enabled??true}onChange={e=>setEditing({...editing,enabled:e.target.checked})}className="rounded"/>Enabled</label>
        <input type="datetime-local"value={editing.start_date?editing.start_date.slice(0,16):""}onChange={e=>setEditing({...editing,start_date:e.target.value?new Date(e.target.value).toISOString():null})}className="border rounded-lg px-3 py-2 text-sm"/>
        <input type="datetime-local"value={editing.end_date?editing.end_date.slice(0,16):""}onChange={e=>setEditing({...editing,end_date:e.target.value?new Date(e.target.value).toISOString():null})}className="border rounded-lg px-3 py-2 text-sm"/>
      </div>
      <div className="flex justify-end gap-2 mt-3"><button onClick={()=>{setEditing(null);setCreating(false);}}className="px-3 py-1.5 text-sm rounded-lg border">Cancel</button><button onClick={save}className="px-3 py-1.5 text-sm rounded-lg text-white"style={{background:"var(--color-brand)"}}>Save</button></div>
    </div>)}
    {isLoading?<div className="text-center py-12 text-slate-400">Loading...</div>:al.length===0?<div className="text-center py-12"><p className="text-slate-400 text-lg mb-2">No banners found</p><button onClick={()=>{setEditing({...EMPTY});setCreating(true);}}className="text-sm"style={{color:"var(--color-brand)"}}>Create your first banner</button></div>:
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">{al.map(ad=>{const s=status(ad);return(<div key={ad.id}className="rounded-xl border border-slate-200 bg-white overflow-hidden">
      {ad.image_url&&<div className="h-32 bg-slate-100 overflow-hidden"><img src={ad.image_url}alt={ad.title}className="w-full h-full object-cover"/></div>}
      <div className="p-4"><div className="flex items-start justify-between mb-2"><div><h4 className="font-semibold text-sm">{ad.title}</h4><p className="text-xs text-slate-500">{plLabel(ad.placement)}&middot;Pos {ad.position}</p></div><StatusBadge status={s}variant={s==="active"?"success":s==="scheduled"?"info":"neutral"}/></div>
      <div className="flex items-center gap-4 text-xs text-slate-500 mb-3"><span className="flex items-center gap-1"><Eye className="w-3 h-3"/>{ad.view_count}</span><span className="flex items-center gap-1"><MousePointerClick className="w-3 h-3"/>{ad.click_count}</span></div>
      <div className="flex items-center gap-1"><button onClick={()=>tMut.mutate(ad.id)}className={`p-1.5 rounded text-xs font-medium ${ad.enabled?"text-emerald-600 hover:bg-emerald-50":"text-slate-400 hover:bg-slate-50"}`}>{ad.enabled?"Disable":"Enable"}</button><button onClick={()=>{setEditing(ad);setCreating(false);}}className="p-1.5 hover:bg-slate-100 rounded"><Pencil className="w-3.5 h-3.5 text-slate-500"/></button><button onClick={()=>{if(confirm("Delete?"))dMut.mutate(ad.id);}}className="p-1.5 hover:bg-red-50 rounded"><Trash2 className="w-3.5 h-3.5 text-red-400"/></button></div></div></div>);})}</div>}
  </div>);
}
