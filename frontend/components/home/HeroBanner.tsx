'use client'
import Link from 'next/link';
import { useState, useEffect, useCallback } from "react";
import { ChevronLeft, ChevronRight } from "lucide-react";

const SLIDES = [
  {
    id: 1,
    headline: "Top tech for your ride",
    sub: "Explore in-car entertainment, GPS, and security devices.",
    cta: "Shop now",
    href: "/listings?category=electronics",
    bg: "#f2f8fd",
    accent: "#0071CE",
    products: [
      { seed: "car-screen-dash", label: "Entertainment" },
      { seed: "gps-device-nav", label: "GPS" },
      { seed: "security-cam-car", label: "Security" },
    ],
  },
  {
    id: 2,
    headline: "Luxury watches & jewelry",
    sub: "Authentic timepieces from trusted GCC sellers.",
    cta: "Explore now",
    href: "/listings?category=jewelry",
    bg: "#fff8e7",
    accent: "#B8860B",
    products: [
      { seed: "rolex-luxury-gold", label: "Watches" },
      { seed: "diamond-fine-ring", label: "Rings" },
      { seed: "gold-necklace-gcc", label: "Necklaces" },
    ],
  },
  {
    id: 3,
    headline: "Find your next vehicle",
    sub: "Thousands of cars across UAE, KSA, Kuwait and more.",
    cta: "Browse vehicles",
    href: "/listings?category=vehicles",
    bg: "#f0f7ff",
    accent: "#0058A3",
    products: [
      { seed: "toyota-sedan-2023", label: "Sedans" },
      { seed: "suv-luxury-dubai", label: "SUVs" },
      { seed: "sports-coupe-gcc", label: "Sports" },
    ],
  },
  {
    id: 4,
    headline: "Latest smartphones & laptops",
    sub: "New and certified electronics at the best GCC prices.",
    cta: "Shop electronics",
    href: "/listings?category=electronics",
    bg: "#f5f0ff",
    accent: "#6D28D9",
    products: [
      { seed: "iphone15-space-black", label: "iPhones" },
      { seed: "macbook-pro-silver", label: "MacBooks" },
      { seed: "samsung-galaxy-ultra", label: "Samsung" },
    ],
  },
];

export function HeroBanner() {
  const [current, setCurrent] = useState(0);
  const [paused, setPaused] = useState(false);

  const next = useCallback(() => setCurrent((c) => (c + 1) % SLIDES.length), []);
  const prev = useCallback(() => setCurrent((c) => (c - 1 + SLIDES.length) % SLIDES.length), []);

  useEffect(() => {
    if (paused) return;
    const t = setInterval(next, 5000);
    return () => clearInterval(t);
  }, [paused, next]);

  const slide = SLIDES[current];

  return (
    <div
      className="relative overflow-hidden transition-colors duration-700"
      style={{ backgroundColor: slide.bg }}
      onMouseEnter={() => setPaused(true)}
      onMouseLeave={() => setPaused(false)}
    >
      <div className="max-w-7xl mx-auto px-6 sm:px-10 py-10 md:py-12 flex flex-col md:flex-row items-center gap-8 min-h-[260px]">

        {/* Left */}
        <div className="flex-1 min-w-0">
          <p className="text-xs font-bold uppercase tracking-widest mb-2" style={{ color: slide.accent }}>
            Mnbarh Deals
          </p>
          <h2 className="text-3xl md:text-4xl font-black text-gray-900 leading-tight">
            {slide.headline}
          </h2>
          <p className="text-gray-500 mt-3 text-base max-w-sm leading-relaxed">
            {slide.sub}
          </p>
          <Link
            href={slide.href}
            className="mt-6 inline-block text-white px-7 py-2.5 rounded-full font-bold text-sm transition-colors shadow-md hover:opacity-90"
            style={{ backgroundColor: slide.accent }}
          >
            {slide.cta}
          </Link>
        </div>

        {/* Right: 3 product images */}
        <div className="hidden md:flex gap-6 shrink-0">
          {slide.products.map((p) => (
            <Link key={p.seed} href={slide.href} className="flex flex-col items-center gap-2.5 group">
              <div className="w-40 h-32 rounded-2xl overflow-hidden bg-white shadow-md group-hover:shadow-xl transition-all duration-300 ring-1 ring-gray-100">
                <img
                  src={`https://picsum.photos/seed/${p.seed}/240/180`}
                  alt={p.label}
                  className="w-full h-full object-cover group-hover:scale-105 transition-transform duration-400"
                />
              </div>
              <span className="text-sm font-semibold text-gray-700 group-hover:text-[#0071CE] transition-colors flex items-center gap-1">
                {p.label} <ChevronRight size={13} />
              </span>
            </Link>
          ))}
        </div>
      </div>

      {/* Arrows */}
      <button
        onClick={prev}
        className="absolute left-3 top-1/2 -translate-y-1/2 w-9 h-9 rounded-full bg-white shadow-md flex items-center justify-center text-gray-600 hover:text-[#0071CE] hover:shadow-lg transition-all border border-gray-100"
      >
        <ChevronLeft size={20} />
      </button>
      <button
        onClick={next}
        className="absolute right-3 top-1/2 -translate-y-1/2 w-9 h-9 rounded-full bg-white shadow-md flex items-center justify-center text-gray-600 hover:text-[#0071CE] hover:shadow-lg transition-all border border-gray-100"
      >
        <ChevronRight size={20} />
      </button>

      {/* Dots */}
      <div className="absolute bottom-3 left-1/2 -translate-x-1/2 flex gap-2">
        {SLIDES.map((_, i) => (
          <button
            key={i}
            onClick={() => setCurrent(i)}
            className="h-2 rounded-full transition-all duration-300"
            style={{
              width: i === current ? "20px" : "8px",
              backgroundColor: i === current ? slide.accent : "#CBD5E1",
            }}
          />
        ))}
      </div>
    </div>
  );
}
