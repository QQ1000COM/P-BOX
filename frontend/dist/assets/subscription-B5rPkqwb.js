import{c as i,h as t}from"./index-BHMT_Ysd.js";/**
 * @license lucide-react v0.487.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const o=[["path",{d:"M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71",key:"1cjeqo"}],["path",{d:"M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71",key:"19qd67"}]],c=i("link",o),a={list:()=>t.get("/subscriptions"),get:s=>t.get(`/subscriptions/${s}`),getNodes:s=>t.get(`/subscriptions/${s}/nodes`),add:s=>t.post("/subscriptions",s),update:(s,e)=>t.put(`/subscriptions/${s}`,e),delete:s=>t.delete(`/subscriptions/${s}`),refresh:s=>t.post(`/subscriptions/${s}/update`),refreshAll:()=>t.post("/subscriptions/update-all")};export{c as L,a as s};
