//! Byline: Claude Code · Opus 5 · 2026-09-17
//!
//! `#[command]` leaves the donor function untouched and adds a braced struct with
//! the same name (type namespace only, so `pub use module::cmd` re-exports both).
//! The struct implements `tauri::Command`, which pulls each argument out of the
//! JSON body exactly the way Tauri's IPC does (camelCase key, snake_case
//! fallback), injects `AppHandle` / `State<T>` / `Window`, awaits async bodies and
//! serialises `Ok` / `Err` the way the web-mode contract expects.
//! `generate_handler![a::b, c::d]` expands to a `Vec<(&str, CommandFn)>`.

use proc_macro::TokenStream;
use proc_macro2::Span;
use quote::{format_ident, quote};
use syn::parse::{Parse, ParseStream};
use syn::punctuated::Punctuated;
use syn::{parse_macro_input, FnArg, Ident, ItemFn, Pat, Path, ReturnType, Token, Type};

fn camel(name: &str) -> String {
    let mut out = String::new();
    let mut upper = false;
    for (i, ch) in name.chars().enumerate() {
        if ch == '_' && i > 0 {
            upper = true;
        } else if upper {
            out.extend(ch.to_uppercase());
            upper = false;
        } else {
            out.push(ch);
        }
    }
    out
}

fn last_ident(ty: &Type) -> Option<String> {
    match ty {
        Type::Path(p) => p.path.segments.last().map(|s| s.ident.to_string()),
        Type::Reference(r) => last_ident(&r.elem),
        _ => None,
    }
}

#[proc_macro_attribute]
pub fn command(_attr: TokenStream, item: TokenStream) -> TokenStream {
    let func = parse_macro_input!(item as ItemFn);
    let name = &func.sig.ident;
    let vis = &func.vis;
    let is_async = func.sig.asyncness.is_some();
    let mut lets = Vec::new();
    let mut call_args = Vec::new();

    for (idx, input) in func.sig.inputs.iter().enumerate() {
        let FnArg::Typed(pt) = input else {
            return syn::Error::new_spanned(input, "self commands are not supported")
                .to_compile_error()
                .into();
        };
        let raw = match &*pt.pat {
            Pat::Ident(pi) => pi.ident.to_string(),
            _ => format!("arg{idx}"),
        };
        let key = raw.trim_start_matches('_').to_string();
        let var = format_ident!("__a{}", idx);
        let ty = &pt.ty;
        match last_ident(ty).as_deref() {
            Some("AppHandle") => lets.push(quote! { let #var = __ctx.app.clone(); }),
            Some("Window") | Some("WebviewWindow") => {
                lets.push(quote! { let #var = __ctx.app.window(); })
            }
            Some("State") => lets.push(quote! { let #var: #ty = __ctx.app.state_for_command()?; }),
            _ => {
                let camel_key = camel(&key);
                lets.push(quote! {
                    let #var: #ty = __ctx.arg(#camel_key, #key)?;
                });
            }
        }
        call_args.push(var);
    }

    let returns_result = match &func.sig.output {
        ReturnType::Default => None,
        ReturnType::Type(_, ty) => Some(
            last_ident(ty)
                .map(|n| n.ends_with("Result"))
                .unwrap_or(false),
        ),
    };

    let invoke = if is_async {
        quote! { #name(#(#call_args),*).await }
    } else {
        quote! { ::tauri::__private::block_in_place(|| #name(#(#call_args),*)) }
    };
    let convert = match returns_result {
        None => quote! { { let _ = __r; Ok(::tauri::__private::Value::Null) } },
        Some(true) => quote! { ::tauri::__private::from_result(__r) },
        Some(false) => quote! { ::tauri::__private::from_value(__r) },
    };

    let expanded = quote! {
        #func

        #[doc(hidden)]
        #[allow(non_camel_case_types, dead_code)]
        #vis struct #name {}

        #[allow(unused_variables, unused_mut, clippy::all)]
        impl ::tauri::Command for #name {
            fn call(__ctx: ::tauri::CommandCtx) -> ::tauri::CommandFuture {
                ::std::boxed::Box::pin(async move {
                    let __ctx = __ctx;
                    #(#lets)*
                    let __r = #invoke;
                    #convert
                })
            }
        }
    };
    expanded.into()
}

struct Handlers(Punctuated<Path, Token![,]>);

impl Parse for Handlers {
    fn parse(input: ParseStream) -> syn::Result<Self> {
        Ok(Handlers(Punctuated::parse_terminated(input)?))
    }
}

#[proc_macro]
pub fn generate_handler(input: TokenStream) -> TokenStream {
    let Handlers(paths) = parse_macro_input!(input as Handlers);
    let entries = paths.iter().map(|p| {
        let name = p
            .segments
            .last()
            .map(|s| s.ident.to_string())
            .unwrap_or_default();
        let lit = syn::LitStr::new(&name, Span::call_site());
        quote! { (#lit, <#p as ::tauri::Command>::call as ::tauri::CommandFn) }
    });
    let _unused: Option<Ident> = None;
    quote! { ::std::vec![ #(#entries),* ] }.into()
}
