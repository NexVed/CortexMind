import { Component, JSX, createContext, createSignal, useContext, onMount } from 'solid-js';

export interface CortexUser { id: string; email: string; displayName: string; githubUsername: string; githubAvatarUrl: string; githubId: string; provider: string; offline: boolean; }
interface AuthContextValue { user: () => CortexUser | null; token: () => string; isAuthenticated: () => boolean; isLoading: () => boolean; error: () => string; loginWithGitHub: () => Promise<void>; continueOffline: (displayName: string) => Promise<void>; logout: () => Promise<void>; }
const AuthContext = createContext<AuthContextValue>();
const API = import.meta.env.VITE_API_URL || '';
export function useAuth(): AuthContextValue { const ctx=useContext(AuthContext); if(!ctx) throw new Error('useAuth must be used within an AuthProvider'); return ctx; }
function asUser(value: any): CortexUser { return { id:value.id || value.ID || '', email:value.email || value.Email || '', displayName:value.display_name || value.displayName || value.DisplayName || value.username || value.Username || 'Offline', githubUsername:value.username || value.Username || '', githubAvatarUrl:value.avatar_url || value.avatarUrl || value.AvatarURL || '', githubId:value.github_id || value.githubId || value.GitHubID || '', provider:value.provider || value.Provider || '', offline:!!(value.offline ?? value.Offline) }; }
async function request(path:string, init?:RequestInit):Promise<any>{const r=await fetch(API+path,{headers:{'Content-Type':'application/json'},...init});const body=await r.json();if(!r.ok)throw new Error(body.error||'Request failed');return body;}
export const AuthProvider: Component<{children: JSX.Element}> = (props) => {
  const [user,setUser]=createSignal<CortexUser|null>(null); const [isLoading,setLoading]=createSignal(true); const [error,setError]=createSignal('');
  const restore=async()=>{try{const body=await request('/api/auth/session');setUser(body.user ? asUser(body.user) : null)}catch(err:any){setError(err.message)}finally{setLoading(false)}};
  onMount(()=>{void restore()});
  const loginWithGitHub=async()=>{setError('');setLoading(true);try{const {url}=await request('/api/auth/github/start',{method:'POST'});window.open(url,'_blank','noopener,noreferrer');const started=Date.now();const poll=async():Promise<void>=>{await new Promise(r=>setTimeout(r,1000));await restore();if(user()||Date.now()-started>300000)return;return poll()};await poll();if(!user())throw new Error('GitHub sign-in timed out. Please try again.')}catch(err:any){setError(err.message);throw err}finally{setLoading(false)}};
  const continueOffline=async(displayName:string)=>{setError('');setLoading(true);try{const body=await request('/api/auth/offline',{method:'POST',body:JSON.stringify({display_name:displayName})});setUser(asUser(body.user))}catch(err:any){setError(err.message);throw err}finally{setLoading(false)}};
  const logout=async()=>{await request('/api/auth/logout',{method:'POST'});setUser(null)};
  return <AuthContext.Provider value={{user,token:()=>'',isAuthenticated:()=>user()!==null,isLoading,error,loginWithGitHub,continueOffline,logout}}>{props.children}</AuthContext.Provider>;
};
